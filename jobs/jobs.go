package jobs // import "github.com/cisocrgroup/services/pcwgo/jobs"

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"sync"

	"github.com/UNO-SOFT/ulog"
	"github.com/finkf/pcwgo/api"
	"github.com/finkf/pcwgo/db"
)

var (
	js *j
)

type j struct {
	db          db.DB                      // database
	wg          sync.WaitGroup             // wait group for running jobs and stop signal
	queue       chan s                     // jobs queue
	cancelFuncs map[int]context.CancelFunc // active jobs cancel functions
	once        sync.Once                  // used to handle multiple calls to close
}

type s struct {
	id   int
	err  error
	r    Runner
	ctx  context.Context
	stop bool
}

// Init initializes the jobs queue and the jobs database tables (if
// they do not yet exist).  It must be called before any other
// functions in this package and must not called concurrently with any
// other functions in this package.
func Init(dtb db.DB) error {
	if err := db.CreateTableJobs(dtb); err != nil {
		return fmt.Errorf("cannot initialize jobs: %v", err)
	}
	js = &j{
		queue:       make(chan s),
		cancelFuncs: make(map[int]context.CancelFunc),
		db:          dtb,
	}
	go func() { jobs() }()
	return nil
}

// Close closes the jobs queue.  It is save to call it multiple times.
func Close() error {
	js.once.Do(func() {
		// send stop signal to jobs() to cancel all running jobs
		js.queue <- s{stop: true}
		// wait until all running jobs have stoped
		js.wg.Wait()
		ulog.Write("all jobs have been handled")
		// now close the queue
		close(js.queue)
	})
	return nil
}

// Runner defines the interface for any running job
type Runner interface {
	BookID() int               // returns the book id of the job
	Name() string              // returns the name of the job
	Run(context.Context) error // runs the job
}

// Start runs the given callback function as a background job.  It
// starts the job in the background and immediately returns the job id
// without blocking.  If a job for the given book is already running,
// this job's id information is returned.  You can check the status of
// the job with the Job function at any given time.
func Start(ctx context.Context, r Runner) (int, error) {
	job, ok, err := db.FindJobByID(js.db, r.BookID())
	if err != nil {
		return 0, fmt.Errorf("cannot start job id %d: %v", r.BookID(), err)
	}
	if ok && job.StatusID == db.StatusIDRunning {
		return Job(r.BookID()).JobID, nil
	}
	var id int
	if ok {
		if err := db.SetJobStatusWithText(js.db, job.JobID, db.StatusIDRunning, r.Name()); err != nil {
			return 0, fmt.Errorf("cannot start job id %d: %v", job.JobID, err)
		}
		id = job.JobID
	} else {
		xid, err := db.NewJob(js.db, r.BookID(), r.Name())
		if err != nil {
			return 0, fmt.Errorf("cannot start job id %d: %v", r.BookID(), err)
		}
		id = xid
	}
	js.queue <- s{id: id, r: r, ctx: ctx}
	return id, nil
}

// Job returns information about the job with the given id.  If the
// job cannot be found or if any other error occurs, a job with
// db.StatusFailed is returned.
func Job(id int) *api.JobStatus {
	job, ok, err := db.FindJobByID(js.db, id)
	if err != nil {
		ulog.Write("cannot query for job", "id", id, "err", err)
		return &api.JobStatus{StatusID: db.StatusIDFailed, StatusName: db.StatusFailed}
	}
	if !ok {
		ulog.Write("cannot query for job", "id", id, "err", "no such job id")
		return &api.JobStatus{StatusID: db.StatusIDFailed, StatusName: db.StatusFailed}
	}
	return job
}

func jobs() {
	for job := range js.queue {
		if job.r != nil {
			ulog.Write("handling job", "id", job.id, "err", job.err, "runner", job.r.Name())
		} else {
			ulog.Write("handling job", "id", job.id, "err", job.err)
		}
		// we are done: cancel all running jobs
		if job.stop {
			for _, cancel := range js.cancelFuncs {
				cancel()
			}
			continue
		}
		// new job: start it
		if job.r != nil {
			ctx, cancel := context.WithCancel(job.ctx)
			js.cancelFuncs[job.id] = cancel
			r := job.r // must copy function
			id := job.id
			js.wg.Add(1)
			go func() {
				defer js.wg.Done()
				js.queue <- s{id: id, err: r.Run(ctx)}
				ulog.Write("job done", "id", id)
			}()
			continue
		}
		// finished job: handle result and status accordingly
		delete(js.cancelFuncs, job.id)
		if job.err != nil {
			ulog.Write("job failed", "id", job.id, "err", job.err)
			if err := db.SetJobStatus(js.db, job.id, db.StatusIDFailed); err != nil {
				ulog.Write("cannot set job status", "status", db.StatusFailed, "err", err)
			}
			continue
		}
		if err := db.SetJobStatus(js.db, job.id, db.StatusIDDone); err != nil {
			ulog.Write("cannot set job status", "status", db.StatusFailed, "err", err)
		}
	}
	ulog.Write("queue closed")
}

// Run executes a command with the given context and arguments.  The
// command's stderr is logged using log.Debug.  Run waits for the
// command to finish and returns its result.
func Run(ctx context.Context, cmd string, args ...string) error {
	ulog.Write("running command", "cmd", cmd, "args", args)
	exe := exec.CommandContext(ctx, cmd, args...)
	stderr, err := exe.StderrPipe()
	if err != nil {
		return fmt.Errorf("cannot connect to command's stderr: %v", err)
	}
	go func() {
		defer stderr.Close()
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			ulog.Write("stderr", "text", s.Text())
		}
		// we do not care about errors
	}()
	// we ignore command's stdout
	if err := exe.Run(); err != nil {
		return fmt.Errorf("cannot run command: %v", err)
	}
	return nil
}

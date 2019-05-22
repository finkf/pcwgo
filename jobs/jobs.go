package jobs // import "github.com/finkf/pcwgo/jobs"

import (
	"context"
	"fmt"
	"sync"

	"github.com/finkf/pcwgo/api"
	"github.com/finkf/pcwgo/db"
	log "github.com/sirupsen/logrus"
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
	f    Func
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
		log.Infof("all jobs have been handled")
		// now close the queue
		close(js.queue)
	})
	return nil
}

// Func defines the callback functions for jobs in the queue.
type Func func(context.Context) error

// Start runs the given callback function as a background job.  It
// starts the job in the background and immediately returns the job id
// if the process was started without blocking.  You can check the
// status of the job with the Job function at any given time.
func Start(ctx context.Context, bookID int, f Func) (int, error) {
	job, ok, err := db.FindJobByID(js.db, bookID)
	if err != nil {
		return 0, fmt.Errorf("cannot start job id %d: %v", bookID, err)
	}
	if ok && job.StatusID == db.StatusIDRunning {
		return 0, fmt.Errorf("cannot start job id %d: running", bookID)
	}
	var id int
	if ok {
		if err := setStatus(job.JobID, db.StatusIDRunning, db.StatusRunning); err != nil {
			return 0, fmt.Errorf("cannot start job id %d: %v", bookID, err)
		}
		id = job.JobID
	} else {
		xid, err := db.NewJob(js.db, bookID)
		if err != nil {
			return 0, fmt.Errorf("cannot start job id %d: %v", bookID, err)
		}
		id = xid
	}
	js.queue <- s{id: id, f: f, ctx: ctx}
	return id, nil
}

// Job returns information about the job with the given id.  If the
// job cannot be found or if any other error occurs, a job with
// db.StatusFailed is returned.
func Job(id int) *api.Job {
	job, ok, err := db.FindJobByID(js.db, id)
	if err != nil {
		log.Infof("cannot query for job id %d: %v", id, err)
		return &api.Job{StatusID: db.StatusIDFailed, StatusName: db.StatusFailed}
	}
	if !ok {
		log.Infof("cannot query for job id %d: no such job id", id)
		return &api.Job{StatusID: db.StatusIDFailed, StatusName: db.StatusFailed}
	}
	return job
}

func jobs() {
	for job := range js.queue {
		log.Debugf("job: %v", job)
		// we are done: cancel all running jobs
		if job.stop {
			for _, cancel := range js.cancelFuncs {
				cancel()
			}
			continue
		}
		// new job: start it
		if job.f != nil {
			ctx, cancel := context.WithCancel(job.ctx)
			js.cancelFuncs[job.id] = cancel
			f := job.f // must copy function
			id := job.id
			js.wg.Add(1)
			go func() {
				defer js.wg.Done()
				js.queue <- s{id: id, err: f(ctx)}
				log.Infof("job %d: done", id)
			}()
			continue
		}
		// finished job: handle result and status accordingly
		delete(js.cancelFuncs, job.id)
		if job.err != nil {
			log.Infof("job %d failed: %v", job.id, job.err)
			if err := setStatus(job.id, db.StatusIDFailed, db.StatusFailed); err != nil {
				log.Info(err)
			}
			continue
		}
		if err := setStatus(job.id, db.StatusIDDone, db.StatusDone); err != nil {
			log.Info(err)
		}
	}
	log.Debug("queue closed")
}

func setStatus(id, status int, name string) error {
	if err := db.SetJobStatus(js.db, id, status); err != nil {
		return fmt.Errorf("cannot set job status to %s: %v", name, err)
	}
	return nil
}
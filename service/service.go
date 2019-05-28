package service // import "github.com/finkf/pcwgo/service"

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/bluele/gcache"
	"github.com/finkf/pcwgo/api"
	"github.com/finkf/pcwgo/db"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

// ProjectCacheSize defines the maximal number of entries in the
// project cache.
var ProjectCacheSize = 10

// AuthCacheSize defines the maximal number of entries in the
// authentication cache.
var AuthCacheSize = 10

// MaxRetries defines the number of times wait tries to connect to the
// database.  Zero means unlimited retries.
var MaxRetries = 10

// Wait defines the time to wait between retries for the database
// connection.
var Wait = 2 * time.Second

// internal sql handle
var pool *sql.DB

// Init sets up the database connection pool using the supplied DSN
// `user:pass@proto(host/dbname`.  Init waits for the databsase to be
// online.  It is not save to call Init from different go routines.
func Init(dsn string) error {
	// connect to db
	log.Debugf("connecting to database with %s", dsn)
	dtb, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	pool = dtb
	// pool.SetMaxOpenConns(100)
	// pool.SetConnMaxLifetime(100)
	// pool.SetMaxIdleConns(10)
	// setup caches
	projectCache = gcache.New(ProjectCacheSize).LoaderFunc(loadProject).Build()
	authCache = gcache.New(AuthCacheSize).LoaderFunc(loadSession).Build()
	// wait for the database and return
	return wait()
}

func wait() error {
	for i := 0; MaxRetries == 0 || i < MaxRetries; i++ {
		stmt := "SELECT id FROM users"
		rows, err := db.Query(pool, stmt)
		if err != nil {
			log.Debugf("error connecting to the database: %v", err)
			time.Sleep(Wait)
			continue
		}
		// successfully connected to the database
		rows.Close()
		log.Debugf("connected sucessfully to database")
		return nil
	}
	return fmt.Errorf("failed to connect to database after %d attempts",
		MaxRetries)
}

// Close closes the database pool.
func Close() {
	pool.Close()
}

// Pool returns the database connection pool that was initialized with
// Init.
func Pool() db.DB {
	return pool
}

// Data defines the payload data for request handlers.
type Data struct {
	Session *api.Session // authentification information
	Project *db.Project  // requested project
	Post    interface{}  // post data
	ID      int          // major id
}

// HandlerFunc defines the callback function to handle callbacks with
// data.
type HandlerFunc func(http.ResponseWriter, *http.Request, *Data)

// WithProject loads the book data of the given book id in the url and
// calls the given callback function.
func WithProject(f HandlerFunc) HandlerFunc {
	re := regexp.MustCompile(`/books/(\d+)`)
	return func(w http.ResponseWriter, r *http.Request, d *Data) {
		var id int
		if err := ParseIDs(r.URL.String(), re, &id); err != nil {
			ErrorResponse(w, http.StatusNotFound, "invalid book ID: %s", r.URL)
			return
		}
		project, found, err := getCachedProject(id)
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError,
				"cannot find book ID %d: %v", id, err)
			return
		}
		if !found {
			ErrorResponse(w, http.StatusNotFound,
				"cannot find book ID %d", id)
			return
		}
		d.Project = project
		f(w, r, d)
	}
}

// WithMethods dispatches a given pair of the request methods to the
// given HandlerFunc with an empty data context.  The first element
// must be of type string, the second argument must be of type
// HandlerFunc.
func WithMethods(args ...interface{}) http.HandlerFunc {
	if len(args)%2 != 0 {
		panic("invalid number of arguments")
	}
	methods := make(map[string]HandlerFunc, len(args)%2)
	for i := 0; i < len(args); i += 2 {
		switch t := args[i+1].(type) {
		case HandlerFunc:
			methods[args[i].(string)] = t
		case func(http.ResponseWriter, *http.Request, *Data):
			methods[args[i].(string)] = HandlerFunc(t)
		default:
			log.Fatalf("invalid type in WithMethods: %T", t)
		}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		f, ok := methods[r.Method]
		if !ok {
			ErrorResponse(w, http.StatusMethodNotAllowed,
				"invalid method: %s", r.Method)
			return
		}
		f(w, r, &Data{})
	}
}

// WithAuth checks if the given request contains a valid
// authentication token.  If not an appropriate error is returned
// before the given callback function is called.
func WithAuth(f HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, d *Data) {
		if len(r.URL.Query()["auth"]) != 1 {
			ErrorResponse(w, http.StatusForbidden,
				"cannot authenticate: missing auth parameter")
			return
		}
		auth := r.URL.Query()["auth"][0]
		log.Debugf("authenticating with %s", auth)
		s, found, err := getCachedSession(auth)
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError,
				"cannot authenticate: %v", err)
			return
		}
		if !found {
			ErrorResponse(w, http.StatusForbidden,
				"cannot authenticate: invalid authentification: %s", auth)
			return
		}
		log.Infof("user %s authenticated: %s (expires: %s)",
			s.User, s.Auth, time.Unix(s.Expires, 0).Format(time.RFC3339))
		d.Session = s
		f(w, r, d)
	}
}

// WithLog wraps logging around the handling of the request.
func WithLog(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Infof("handling [%s] %s", r.Method, r.URL)
		f(w, r)
		log.Infof("handled [%s] %s", r.Method, r.URL)
	}
}

// ErrorResponse writes an error response.  It sets the according
// response header and sends a json-formatted response object.
func ErrorResponse(w http.ResponseWriter, s int, f string, args ...interface{}) {
	message := fmt.Sprintf(f, args...)
	log.Debug(message)
	status := http.StatusText(s)
	w.Header().Set("Content-Type", "application/json") // set Content-Type before call to WriteHeader
	w.WriteHeader(s)
	JSONResponse(w, struct {
		Code    int    `json:"code"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}{s, status, message})
}

// JSONResponse writes a json-formatted response.  Any errors are
// being logged.
func JSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Infof("cannot write json response: %v", err)
	}
}

// ParseIDs parses the numeric fields of the given regex into the
// given id pointers.
func ParseIDs(url string, re *regexp.Regexp, ids ...*int) error {
	m := re.FindStringSubmatch(url)
	if len(m) != len(ids)+1 {
		return fmt.Errorf("invalid url: %s", url)
	}
	for i := 1; i < len(m); i++ {
		id, err := strconv.Atoi(m[i])
		if err != nil {
			return fmt.Errorf("cannot convert number: %v", err)
		}
		*ids[i-1] = id
	}
	return nil
}

// IsValidStatus returns true if the given response has any of the given
// status codes.
func IsValidStatus(r *http.Response, codes ...int) bool {
	for _, code := range codes {
		if r.StatusCode == code {
			return true
		}
	}
	return false
}

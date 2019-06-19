package service // import "github.com/finkf/pcwgo/service"

import (
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
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
	Session *api.Session   // authentification information
	Project *db.Project    // requested project
	Post    interface{}    // post data
	IDs     map[string]int // ids
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
		if n := ParseIDs(r.URL.String(), re, &id); n != 1 {
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

// DropProject removes the active project from the cache if it is
// non-nil.  Should be joined after WithProject.
func DropProject(f HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, d *Data) {
		f(w, r, d)
		if d.Project != nil {
			RemoveProject(d.Project)
		}
	}
}

// DropSession removes the active session from the cache if it is
// non-nil.  Should be joined after WithAuth.
func DropSession(f HandlerFunc) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, d *Data) {
		f(w, r, d)
		if d.Session != nil {
			RemoveSession(d.Session)
		}
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

// WithIDs fills the IDs map with the given (key,id) pairs or returns
// status not found if the given ids could not be parsed in the given
// order.
func WithIDs(f HandlerFunc, keys ...string) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, d *Data) {
		ids, err := parseIDMap(r.URL.String(), keys...)
		if err != nil {
			ErrorResponse(w, http.StatusNotFound,
				"cannot parse ids: %v", err)
			return
		}
		d.IDs = ids
		f(w, r, d)
	}
}

// Parse (key,int) pairs from an url `/key/int/...` in the given order
// of keys.
func parseIDMap(url string, keys ...string) (map[string]int, error) {
	res := make(map[string]int, len(keys))
	for _, key := range keys {
		search := "/" + key + "/"
		pos := strings.Index(url, search)
		if pos == -1 {
			return nil, fmt.Errorf("cannot find key: %s", key)
		}
		id, rest, err := parseInt(url[pos+len(search):])
		if err != nil {
			return nil, err
		}
		res[key] = id
		url = rest
	}
	return res, nil
}

// Parse an integer from str to the first `/` or to the end of the
// string.
func parseInt(str string) (int, string, error) {
	pos := strings.Index(str, "/")
	if pos == -1 {
		pos = len(str)
	}
	id, err := strconv.Atoi(str[0:pos])
	if err != nil {
		return 0, "", fmt.Errorf("cannot parse id in string: %s", str)
	}
	return id, str[pos:], nil
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

// GZIPJSONResponse writes a gzipped json-formatted response.  Any
// errors are being logged.
func GZIPJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Add("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()
	if err := json.NewEncoder(gz).Encode(data); err != nil {
		log.Infof("cannot write gzipped json response: %v", err)
	}
}

// ParseIDs parses the numeric fields of the given regex into the
// given id pointers.  It returns the number of ids parsed.
func ParseIDs(url string, re *regexp.Regexp, ids ...*int) int {
	m := re.FindStringSubmatch(url)
	var i int
	for i = 0; i < len(ids) && i+1 < len(m); i++ {
		id, err := strconv.Atoi(m[i+1])
		if err != nil {
			return 0
		}
		*ids[i] = id
	}
	return i
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

// LinkOrCopy tries to hard link dest to src.  If the file or link
// already exists nothing is done and no error is returned.  If the
// linking fails, LinkOrCopy tries to copy src to dest.
func LinkOrCopy(src, dest string) (err error) {
	if err := os.Link(src, dest); err == nil || os.IsExist(err) {
		return nil
	}
	// Cannot link -> copy
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("cannot copy file: %v", err)
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("cannot copy file: %v", err)
	}
	defer func() {
		ex := out.Close()
		if err == nil {
			err = ex
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("cannot copy file: %v", err)
	}
	return nil
}

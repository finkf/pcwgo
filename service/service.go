package service // import "github.com/finkf/pcwgo/service"

import (
	"compress/gzip"
	"context"
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

	"github.com/finkf/pcwgo/api"
	"github.com/finkf/pcwgo/db"
	_ "github.com/go-sql-driver/mysql" // to connect with mysql
	log "github.com/sirupsen/logrus"
)

type key int

const (
	authKey key = iota
	projectKey
	userIDKey
	projectIDKey
	pageIDKey
	lineIDKey
	jobIDKey
)

// AuthFromCtx returns the registered session from a context.
func AuthFromCtx(ctx context.Context) *api.Session {
	return ctx.Value(authKey).(*api.Session)
}

// ProjectFromCtx returns the registered project from a context.
func ProjectFromCtx(ctx context.Context) *db.Project {
	return ctx.Value(projectKey).(*db.Project)
}

// UserIDFromCtx returns the registered user ID from a context.
func UserIDFromCtx(ctx context.Context) int {
	return ctx.Value(userIDKey).(int)
}

// ProjectIDFromCtx returns the registered project ID from a context.
func ProjectIDFromCtx(ctx context.Context) int {
	return ctx.Value(projectIDKey).(int)
}

// PageIDFromCtx returns the registered page ID from a context.
func PageIDFromCtx(ctx context.Context) int {
	return ctx.Value(pageIDKey).(int)
}

// LineIDFromCtx returns the registered line ID from a context.
func LineIDFromCtx(ctx context.Context) int {
	return ctx.Value(lineIDKey).(int)
}

// JobIDFromCtx returns the registered job ID from a context.
func JobIDFromCtx(ctx context.Context) int {
	return ctx.Value(jobIDKey).(int)
}

// MaxRetries defines the number of times wait tries to connect to the
// database.  Zero means unlimited retries.
var MaxRetries = 10

// Wait defines the time to wait between retries for the database
// connection.
var Wait = 2 * time.Second

// internal sql handle
var pool *sql.DB

// InitDebug sets up the mysql database connection pool using the
// supplied DSN `user:pass@proto(host/dbname)` and sets the log level
// to debug if debug=true.  It then calls Init(dsn) and returns its
// result.
func InitDebug(dsn string, debug bool) error {
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	return Init(dsn)
}

// Init sets up the mysql database connection pool using the supplied
// DSN `user:pass@proto(host/dbname)`.  Init waits for the databsase
// to be online.  It is not save to call Init from different go
// routines.
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

	// wait for the database and return
	return wait(MaxRetries, Wait)
}

func wait(retries int, sleep time.Duration) error {
	for i := 0; retries == 0 || i < retries; i++ {
		rows, err := db.Query(pool, "SELECT id FROM users")
		if err != nil {
			log.Debugf("error connecting to the database: %v", err)
			time.Sleep(sleep)
			continue
		}
		// successfully connected to the database
		rows.Close()
		log.Debugf("connected sucessfully to database")
		return nil
	}
	return fmt.Errorf("failed to connect to database after %d attempts", retries)
}

// Close closes the database pool.
func Close() {
	pool.Close()
}

// Pool returns the database connection pool that was initialized with
// Init.
func Pool() *sql.DB {
	return pool
}

// HandlerFunc defines the callback function to handle callbacks with
// data.
type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

// WithProject loads the project data for the given project id and
// puts it into the context.  It can be retrieved with
// ProjectFromCtx(ctx).
func WithProject(f HandlerFunc) HandlerFunc {
	re := regexp.MustCompile(`/books/(\d+)`)
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var id int
		if n := ParseIDs(r.URL.String(), re, &id); n != 1 {
			ErrorResponse(w, http.StatusNotFound, "cannot find project ID: %s", r.URL)
			return
		}
		p, found, err := db.FindProjectByID(pool, id)
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError,
				"cannot find project ID %d: %v", id, err)
			return
		}
		if !found {
			ErrorResponse(w, http.StatusNotFound,
				"cannot find project ID %d", id)
			return
		}
		f(context.WithValue(ctx, projectKey, p), w, r)
	}
}

// WithMethods dispatches a given pair of the request methods to the
// given HandlerFunc with an empty data context.  The first element
// must be of type string, the second argument must be of type
// HandlerFunc.  The function panics if it encounteres an invalid
// type.
func WithMethods(args ...interface{}) http.HandlerFunc {
	if len(args)%2 != 0 {
		panic("invalid number of arguments")
	}
	methods := make(map[string]HandlerFunc, len(args)%2)
	for i := 0; i < len(args); i += 2 {
		method := args[i].(string) // must be a string
		switch t := args[i+1].(type) {
		case HandlerFunc:
			methods[method] = t
		case func(context.Context, http.ResponseWriter, *http.Request):
			methods[method] = HandlerFunc(t)
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
		f(context.Background(), w, r)
	}
}

// WithUserID extracts the "/users/<numeric id>" part from the url,
// loads it and puts it into the context.  The value can be retrieved
// with UserIDFromCtx(ctx).
func WithUserID(f HandlerFunc) HandlerFunc {
	re := regexp.MustCompile(`/users/(\d+)`)
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var userID int
		if n := ParseIDs(r.URL.String(), re, &userID); n != 1 {
			ErrorResponse(w, http.StatusBadRequest,
				"cannot find user ID: %s", r.URL.String())
			return
		}
		f(context.WithValue(ctx, userIDKey, userID), w, r)
	}
}

// WithProjectID extracts the "/books/<numeric id>" part from the url,
// loads it and puts it into the context.  The value can be retrieved
// with ProjectIDFromCtx(ctx).
func WithProjectID(f HandlerFunc) HandlerFunc {
	re := regexp.MustCompile(`/books/(\d+)`)
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var projectID int
		if n := ParseIDs(r.URL.String(), re, &projectID); n != 1 {
			ErrorResponse(w, http.StatusBadRequest,
				"cannot find project ID: %s", r.URL.String())
			return
		}
		f(context.WithValue(ctx, projectIDKey, projectID), w, r)
	}
}

// WithPageID extracts the "/pages/<numeric id>" part from the url,
// loads it and puts it into the context.  The value can be retrieved
// with PageIDFromCtx(ctx).
func WithPageID(f HandlerFunc) HandlerFunc {
	re := regexp.MustCompile(`/pages/(\d+)`)
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var pageID int
		if n := ParseIDs(r.URL.String(), re, &pageID); n != 1 {
			ErrorResponse(w, http.StatusBadRequest,
				"cannot find page ID: %s", r.URL.String())
			return
		}
		f(context.WithValue(ctx, pageIDKey, pageID), w, r)
	}
}

// WithLineID extracts the "/lines/<numeric id>" part from the url,
// loads it and puts it into the context.  The value can be retrieved
// with LineIDFromCtx(ctx).
func WithLineID(f HandlerFunc) HandlerFunc {
	re := regexp.MustCompile(`/lines/(\d+)`)
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var lineID int
		if n := ParseIDs(r.URL.String(), re, &lineID); n != 1 {
			ErrorResponse(w, http.StatusBadRequest,
				"cannot find line ID: %s", r.URL.String())
			return
		}
		f(context.WithValue(ctx, lineIDKey, lineID), w, r)
	}
}

// WithJobID extracts the "/lines/<numeric id>" part from the url,
// loads it and puts it into the context.  The value can be retrieved
// with JobIDFromCtx(ctx).
func WithJobID(f HandlerFunc) HandlerFunc {
	re := regexp.MustCompile(`/jobs/(\d+)`)
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var jobID int
		if n := ParseIDs(r.URL.String(), re, &jobID); n != 1 {
			ErrorResponse(w, http.StatusBadRequest,
				"cannot find job ID: %s", r.URL.String())
			return
		}
		f(context.WithValue(ctx, jobIDKey, jobID), w, r)
	}
}

// GetIDs parses the given url for (key/int) pairs and put the
// resulting ids into the map.  If a map key start with a "?" the
// given id is optional.  If all keys can be found the function return
// true or false if a key is missing.  Missing optional keys do not
// cause this function to return false if the key/id pair is missing.
func GetIDs(ids map[string]int, url string) bool {
	for key := range ids {
		var opt bool
		if strings.HasPrefix(key, "?") {
			opt = true
			delete(ids, key)
			key = key[1:]
		}
		search := "/" + key + "/"
		pos := strings.Index(url, search)
		if pos == -1 && !opt {
			return false
		}
		if pos == -1 && opt {
			continue
		}
		id, _, err := parseInt(url[pos+len(search):])
		if err != nil {
			return false
		}
		ids[key] = id
	}
	return true
}

// Parse an integer from str to the first `/` or to the end of the
// string.
func parseInt(str string) (int, string, error) {
	pos := strings.IndexAny(str, "/?")
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
// before the given callback function is called.  If the
// authentification succeeds, the session is put into the context and
// can be retrieved using AuthFromCtx(ctx).
func WithAuth(f HandlerFunc) HandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Query()["auth"]) != 1 {
			ErrorResponse(w, http.StatusUnauthorized,
				"cannot authenticate: missing auth parameter")
			return
		}
		auth := r.URL.Query()["auth"][0]
		log.Debugf("authenticating with %s", auth)
		s, found, err := db.FindSessionByID(pool, auth)
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError,
				"cannot authenticate: %v", err)
			return
		}
		if !found {
			ErrorResponse(w, http.StatusUnauthorized,
				"cannot authenticate: invalid authentification")
			return
		}
		log.Infof("user %s authenticated: %s (expires: %s)",
			s.User, s.Auth, time.Unix(s.Expires, 0).Format(time.RFC3339))
		if s.Expired() {
			ErrorResponse(w, http.StatusUnauthorized,
				"cannot authenticate: session expired: %s",
				time.Unix(s.Expires, 0).Format(time.RFC3339))
			return
		}
		f(context.WithValue(ctx, authKey, s), w, r)
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
	log.Infof("error: %s", message)
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

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
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

// MaxRetries defines the number of times wait tries to connect to the
// database.  Zero means unlimited retries.
var MaxRetries = 10

// Wait defines the time to wait between retries for the database
// connection.
var Wait = 2 * time.Second

// internal sql handle
var pool *sql.DB

// InitDebug sets up the database connection pool using the supplied
// DSN `user:pass@proto(host/dbname` and sets the log level to debug
// if debug=true.  It then calls Init(dsn) and returns its result.
func InitDebug(dsn string, debug bool) error {
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	return Init(dsn)
}

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
type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request)

// WithProject loads the project data for the given project id and
// puts it into the context using "project" as key.  Then it calls the
// given handler function.
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
		f(context.WithValue(ctx, "project", p), w, r)
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

// WithUser extracts the "/users/<numeric id>" part from the url,
// loads it and puts it into the context with the key "user".
func WithUser(f HandlerFunc) HandlerFunc {
	re := regexp.MustCompile(`/users/(\d+)`)
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		var userID int
		if n := ParseIDs(r.URL.String(), re, &userID); n != 1 {
			ErrorResponse(w, http.StatusNotFound,
				"cannot find: %s", r.URL.String())
			return
		}
		user, found, err := db.FindUserByID(pool, int64(userID))
		if err != nil {
			ErrorResponse(w, http.StatusInternalServerError,
				"cannot lookup user: %v", err)
			return
		}
		if !found {
			ErrorResponse(w, http.StatusNotFound,
				"cannot find user ID: %d", userID)
			return
		}
		f(context.WithValue(ctx, "user", user), w, r)
	}
}

// WithIDs fills the IDs map with the given (key,id) pairs or returns
// status not found if the given ids could not be parsed in the given
// order.  You can prefix a key with `?` to mark it optional (the key
// is then not inserted intot the ID map if it is not present in the
// request's URL).
func WithIDs(f HandlerFunc, keys ...string) HandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		ctx, err := idContext(ctx, r.URL.String(), keys...)
		if err != nil {
			ErrorResponse(w, http.StatusNotFound,
				"cannot parse ids: %v", err)
			return
		}
		f(ctx, w, r)
	}
}

// Parse (key,int) pairs from an url `/key/int/...` in the given
// order of keys and put them into the given context.
func idContext(ctx context.Context, url string, keys ...string) (context.Context, error) {
	for _, key := range keys {
		var opt bool
		if strings.HasPrefix(key, "?") {
			opt = true
			key = key[1:]
		}
		search := "/" + key + "/"
		pos := strings.Index(url, search)
		if pos == -1 && !opt {
			return nil, fmt.Errorf("cannot find required key: %s", key)
		}
		if pos == -1 && opt {
			continue
		}
		id, rest, err := parseInt(url[pos+len(search):])
		if err != nil {
			return nil, err
		}
		ctx = context.WithValue(ctx, key, id)
		url = rest
	}
	return ctx, nil
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
// authentification succeeds, the session is put into the context as
// "auth".
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
		f(context.WithValue(ctx, "auth", s), w, r)
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

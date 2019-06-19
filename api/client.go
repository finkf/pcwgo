package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/finkf/gofiler"
	log "github.com/sirupsen/logrus"
)

// DefaultWebHost returns the default host address of the main
// webserver that serves the images.  It just strips the port (if any)
// from the given host address.
func DefaultWebHost(host string) string {
	if pos := strings.LastIndex(host, ":"); pos != -1 {
		return host[:pos]
	}
	return host
}

// Client implements the api calls for the pcw backend.
// Use Login to initalize the client.
type Client struct {
	client        *http.Client
	Host, WebHost string
	Session       Session // active session
	Skip, Max     int     // skip and max parameters for queries
}

// NewClient creates a new client with the given host (and it default
// web host).
func NewClient(host string) *Client {
	return &Client{
		Host:    host,
		WebHost: DefaultWebHost(host),
		client:  &http.Client{},
	}
}

// Authenticate creates a new Client from a given auth-token.
func Authenticate(host, authToken string) *Client {
	c := NewClient(host)
	c.Session.Auth = authToken
	return c
}

// Login creates a new Client and authenticates with the given
// username and password.
func Login(host, email, password string) (*Client, error) {
	// tr := &http.Transport{
	// 	TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify},
	// }
	c := NewClient(host)
	// c := &Client{
	// 	client: &http.Client{}, //Transport: tr},
	// 	Host:   host,
	// }
	login := LoginRequest{
		Email:    email,
		Password: password,
	}
	var s Session
	err := c.post(c.url("/login"), login, &s)
	if err != nil {
		return nil, err
	}
	log.Debugf("session: %s", s)
	c.Session = s
	return c, nil
}

func (c Client) getLogin() (Session, error) {
	var s Session
	err := c.get(c.url("/login", Auth, c.Session.Auth), &s)
	return s, err
}

// Logout logs the user out.
func (c Client) Logout() error {
	return c.get(c.url("/logout", Auth, c.Session.Auth), nil)
}

// GetUsers returns all users (needs admin rights).
func (c Client) GetUsers() (Users, error) {
	var res Users
	err := c.get(c.url("/users", Auth, c.Session.Auth), &res)
	return res, err
}

// GetUser returns the user with the given id.
func (c Client) GetUser(id int64) (User, error) {
	var res User
	err := c.get(c.url(userPath(id), Auth, c.Session.Auth), &res)
	return res, err
}

// PutUser updates the settings for a user and returns it.
func (c Client) PutUser(u CreateUserRequest) (User, error) {
	var res User
	url := c.url(userPath(u.User.ID), Auth, c.Session.Auth)
	err := c.put(url, u, &res)
	return res, err
}

// PostUser creates a new User and returns it.
func (c Client) PostUser(u CreateUserRequest) (User, error) {
	var res User
	url := c.url("/users", Auth, c.Session.Auth)
	err := c.post(url, u, &res)
	return res, err
}

// GetAPIVersion returns the API version of the pocoweb server.
func (c Client) GetAPIVersion() (Version, error) {
	var res Version
	err := c.get(c.url("/api-version"), &res)
	return res, err
}

// PostZIP uploads a zipped OCR project and returns the new project.
func (c Client) PostZIP(zip io.Reader) (*Book, error) {
	var book Book
	url := c.url("/books", Auth, c.Session.Auth)
	err := c.doPost(url, "application/zip", zip, &book)
	return &book, err
}

// PostBook updates the given Book and returns it.
func (c Client) PostBook(book Book) (*Book, error) {
	url := c.url(bookPath(book.ProjectID), Auth, c.Session.Auth)
	var newBook Book
	err := c.post(url, book, &newBook)
	return &newBook, err
}

// GetBook returns the book with the given id.
func (c Client) GetBook(bookID int) (*Book, error) {
	url := c.url(bookPath(bookID), Auth, c.Session.Auth)
	var book Book
	err := c.get(url, &book)
	return &book, err
}

// GetBooks returns all books of a user.
func (c Client) GetBooks() (*Books, error) {
	url := c.url("/books", Auth, c.Session.Auth)
	var books Books
	err := c.get(url, &books)
	return &books, err
}

// GetPage returns the page with the given ids.
func (c Client) GetPage(bookID, pageID int) (*Page, error) {
	var page Page
	url := c.url(pagePath(bookID, pageID), Auth, c.Session.Auth)
	err := c.get(url, &page)
	return &page, err
}

// DeletePage deletes the given page.
func (c Client) DeletePage(bookID, pageID int) error {
	url := c.url(pagePath(bookID, pageID), Auth, c.Session.Auth)
	return c.delete(url)
}

// GetLine returns the line with the given ids.
func (c Client) GetLine(bookID, pageID, lineID int) (*Line, error) {
	var line Line
	url := c.url(linePath(bookID, pageID, lineID), Auth, c.Session.Auth)
	err := c.get(url, &line)
	return &line, err
}

// PostLine posts new content to the given line.
func (c Client) PostLine(bookID, pageID, lineID int, cor Correction) (*Line, error) {
	var line Line
	url := c.url(linePath(bookID, pageID, lineID), Auth, c.Session.Auth)
	err := c.post(url, cor, &line)
	return &line, err
}

// DeleteLine deletes the given line.
func (c Client) DeleteLine(bookID, pageID, lineID int) error {
	url := c.url(linePath(bookID, pageID, lineID), Auth, c.Session.Auth)
	return c.delete(url)
}

// GetTokens returns the tokens for the given line.
func (c Client) GetTokens(bookID, pageID, lineID int) (Tokens, error) {
	var tokens Tokens
	url := c.url(linePath(bookID, pageID, lineID)+"/tokens", Auth, c.Session.Auth)
	err := c.get(url, &tokens)
	return tokens, err
}

// PostToken posts new content to the given token.
func (c Client) PostToken(bookID, pageID, lineID, tokenID int, cor Correction) (*Token, error) {
	var token Token
	url := c.url(tokenPath(bookID, pageID, lineID, tokenID), Auth, c.Session.Auth)
	err := c.post(url, cor, &token)
	return &token, err
}

// Search searches for one or more tokens.
func (c Client) Search(bookID int, q string, qs ...string) (*SearchResults, error) {
	params := []string{Auth, c.Session.Auth, "p", "0",
		"skip", strconv.Itoa(c.Skip), "max", strconv.Itoa(c.Max), "q", q}
	for _, x := range qs {
		params = append(params, "q", x)
	}
	url := c.url(bookPath(bookID)+"/search", params...)
	var res SearchResults
	err := c.get(url, &res)
	return &res, err
}

// SearchErrorPatterns searches for one or more error patterns.
func (c Client) SearchErrorPatterns(bookID int, q string, qs ...string) (*SearchResults, error) {
	params := []string{Auth, c.Session.Auth, "p", "1",
		"skip", strconv.Itoa(c.Skip), "max", strconv.Itoa(c.Max), "q", q}
	for _, x := range qs {
		params = append(params, "q", x)
	}
	url := c.url(bookPath(bookID)+"/search", params...)
	var res SearchResults
	err := c.get(url, &res)
	return &res, err
}

// GetAdaptiveTokens returns the adaptive tokens for the given book.
func (c Client) GetAdaptiveTokens(bookID int) (AdaptiveTokens, error) {
	url := c.url("/profile/adaptive/"+bookPath(bookID), Auth, c.Session.Auth)
	var res AdaptiveTokens
	err := c.get(url, &res)
	return res, err
}

// Split splits a project.
func (c Client) Split(pid, n int, random bool) (*Books, error) {
	url := c.url(bookPath(pid)+"/split", Auth, c.Session.Auth)
	split := struct {
		N int  `json:"n"`
		R bool `json:"random"`
	}{N: n, R: random}
	var books Books
	err := c.post(url, split, &books)
	return &books, err
}

// Assign assigns a project to another user.
func (c Client) Assign(pid, uid int) error {
	url := c.url(bookPath(pid)+"/assign", Auth, c.Session.Auth,
		"uid", fmt.Sprintf("%d", uid))
	return c.get(url, nil)
}

// Finish reassigns a project back to its original owner.
func (c Client) Finish(pid int) error {
	url := c.url(bookPath(pid)+"/finish", Auth, c.Session.Auth)
	return c.get(url, nil)
}

// PostProfile sends a request to profile the book with the given id.
func (c Client) PostProfile(bookID int, tokens ...string) (Job, error) {
	url := c.url("/profile"+bookPath(bookID), Auth, c.Session.Auth)
	var job Job
	return job, c.post(url, AdditionalLexicon{Tokens: tokens}, &job)
}

// GetJobStatus returns the job status for the given job.
func (c Client) GetJobStatus(jobID int) (*JobStatus, error) {
	url := c.url(jobPath(jobID), Auth, c.Session.Auth)
	var job JobStatus
	return &job, c.get(url, &job)
}

// GetProfile downloads the profile for the given book.
func (c Client) GetProfile(bookID int) (gofiler.Profile, error) {
	url := c.url("/profile"+bookPath(bookID), Auth, c.Session.Auth)
	var profile gofiler.Profile
	return profile, c.get(url, &profile)
}

// QueryProfile returns the suggestions for the given words.
func (c Client) QueryProfile(bookID int, q string, qs ...string) (Suggestions, error) {
	params := []string{Auth, c.Session.Auth, "q", q}
	for _, x := range qs {
		params = append(params, "q", x)
	}
	url := c.url("/profile"+bookPath(bookID), params...)
	var suggestions Suggestions
	return suggestions, c.get(url, &suggestions)
}

// GetPatterns returns the ocr or hist error-patterns for the given book.
func (c Client) GetPatterns(bookID int, ocr bool) (PatternCounts, error) {
	params := []string{Auth, c.Session.Auth, "ocr", strconv.FormatBool(ocr)}
	url := c.url("/profile/patterns"+bookPath(bookID), params...)
	var patterns PatternCounts
	return patterns, c.get(url, &patterns)
}

// QueryPatterns returns the suggestions for the given error patterns.
func (c Client) QueryPatterns(bookID int, ocr bool, q string, qs ...string) (Patterns, error) {
	params := []string{Auth, c.Session.Auth, "ocr", strconv.FormatBool(ocr), "q", q}
	for _, x := range qs {
		params = append(params, "q", x)
	}
	url := c.url("/profile/patterns"+bookPath(bookID), params...)
	var patterns Patterns
	return patterns, c.get(url, &patterns)
}

// GetSuspicious returns the suspicious words for the given book.
func (c Client) GetSuspicious(bookID int) (SuggestionCounts, error) {
	url := c.url("/profile/suspicious"+bookPath(bookID), Auth, c.Session.Auth)
	var counts SuggestionCounts
	return counts, c.get(url, &counts)
}

// PostExtendendLexicon sends a request to create the extendedn
// lexicon for the given book or project.
func (c Client) PostExtendedLexicon(bookID int) (Job, error) {
	url := c.url("/postcorrect/el"+bookPath(bookID), Auth, c.Session.Auth)
	var job Job
	return job, c.post(url, nil, &job)
}

// GetExtendendLexicon returns the extended lexicon for the given book
// or project.
func (c Client) GetExtendedLexicon(bookID int) (ExtendedLexicon, error) {
	url := c.url("/postcorrect/el"+bookPath(bookID), Auth, c.Session.Auth)
	var el ExtendedLexicon
	return el, c.get(url, &el)
}

// PostPostCorrection sends a request to start the automatic post
// correction on the given book with the given extended lexicon
// tokens.
func (c Client) PostPostCorrection(bookID int) (Job, error) {
	url := c.url("/postcorrect/rrdm"+bookPath(bookID), Auth, c.Session.Auth)
	var job Job
	return job, c.post(url, nil, &job)
}

// GetExtendendLexicon returns the post-correction data for the given book.
func (c Client) GetPostCorrection(bookID int) (*PostCorrection, error) {
	url := c.url("/postcorrect/rrdm"+bookPath(bookID), Auth, c.Session.Auth)
	var pc PostCorrection
	return &pc, c.get(url, &pc)
}

// GetOCRModels returns the available ocr models for the given book or
// project.
func (c Client) GetOCRModels(bookID int) (Models, error) {
	url := c.url("/ocr"+bookPath(bookID), Auth, c.Session.Auth)
	var models Models
	return models, c.get(url, &models)
}

// PostOCRModel runs the ocr over the given book or project.  If train
// is true, a model is trained on the ground truth data.
func (c Client) PostOCRModel(bid, pid, lid int, name string) (Job, error) {
	model := Model{Name: name}
	prefix := "/ocr"
	if lid != 0 {
		prefix += linePath(bid, pid, lid)
	} else if pid != 0 {
		prefix += pagePath(bid, pid)
	} else {
		prefix += bookPath(bid)
	}
	url := c.url(prefix, Auth, c.Session.Auth)
	var job Job
	return job, c.post(url, model, &job)
}

// Raw sends a get request to the given path and writes the raw
// response content into the given writer.
func (c Client) Raw(path string, out io.Writer) error {
	url := c.url(path, Auth, c.Session.Auth)
	log.Debugf("GET %s", url)
	res, err := c.client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, err = io.Copy(out, res.Body)
	return err
}

// GetLineImage downloads the line image for the given line.  At this
// point only PNGs are accepted.
func (c Client) GetLineImage(line *Line) (image.Image, error) {
	url := c.WebHost + "/" + line.ImgFile
	log.Debugf("GET %s", url)
	res, err := c.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot get line image: bad response %s", res.Status)
	}
	if res.Header.Get("Content-Type") != "image/png" {
		return nil, fmt.Errorf("cannot get line image: invalid Content-Type: %s",
			res.Header.Get("Content-Type"))
	}
	return png.Decode(res.Body)
}

// Download downloads the zipped book's contents.
func (c Client) Download(pid int) (io.ReadCloser, error) {
	// create archive and get its download destination
	xurl := c.url(filepath.Join(bookPath(pid), "download"), Auth, c.Session.Auth)
	var archive struct {
		Archive string `json:"archive"`
	}
	if err := c.get(xurl, &archive); err != nil {
		return nil, fmt.Errorf("cannot download: %v", err)
	}
	// download archive
	log.Debugf("archive path: %s", archive.Archive)
	xurl = c.WebHost + "/" + archive.Archive +
		"?" + url.PathEscape(Auth) + "=" + url.PathEscape(c.Session.Auth)
	log.Debugf("GET %s", xurl)
	res, err := c.client.Get(xurl)
	if err != nil {
		return nil, fmt.Errorf("cannot download: %v", err)
	}
	// do *not* defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("cannot download: bad response %s", res.Status)
	}
	if res.Header.Get("Content-Type") != "application/zip" {
		res.Body.Close()
		return nil, fmt.Errorf("cannot download: invalid Content-Type: %s",
			res.Header.Get("Content-Type"))
	}
	return res.Body, nil
}

func (c Client) url(path string, keyvals ...string) string {
	var b strings.Builder
	b.WriteString(c.Host)
	b.WriteString(path)
	pre := '?'
	for i := 0; i+1 < len(keyvals); i += 2 {
		b.WriteRune(pre)
		b.WriteString(url.PathEscape(keyvals[i]))
		b.WriteRune('=')
		b.WriteString(url.PathEscape(keyvals[i+1]))
		pre = '&'
	}
	return b.String()
}

func userPath(id int64) string {
	return formatID("/users/%d", id)
}

func bookPath(id int) string {
	return formatID("/books/%d", id)
}

func jobPath(id int) string {
	return formatID("/jobs/%d", id)
}

func pagePath(id, pageid int) string {
	return formatID("/books/%d/pages/%d", id, pageid)
}

func linePath(id, pageid, lineid int) string {
	return formatID("/books/%d/pages/%d/lines/%d", id, pageid, lineid)
}

func tokenPath(id, pageid, lineid, tokenid int) string {
	return formatID("/books/%d/pages/%d/lines/%d/tokens/%d",
		id, pageid, lineid, tokenid)
}

func formatID(url string, args ...interface{}) string {
	return fmt.Sprintf(url, args...)
}

func (c Client) get(url string, out interface{}) error {
	log.Debugf("GET %s", url)
	res, err := c.client.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if !valid(res) {
		return doError(res)
	}
	log.Debugf("reponse from server: %s", res.Status)
	if out == nil {
		return nil
	}
	return decodeJSONMaybeGzipped(res, out)
}

func decodeJSONMaybeGzipped(res *http.Response, out interface{}) error {
	var r io.Reader = res.Body
	if res.Header.Get("Content-Encoding") == "gzip" {
		log.Debugf("unzipping content")
		gzip, err := gzip.NewReader(r)
		if err != nil {
			return err
		}
		r = gzip
	}
	err := json.NewDecoder(r).Decode(out)
	if err != nil {
		return fmt.Errorf("cannot decode server response: %v", err)
	}
	return nil
}

func (c Client) delete(url string) error {
	log.Debugf("DELETE %s", url)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("bad response: %s", res.Status)
	}
	return nil
}

func (c Client) put(url string, data, out interface{}) error {
	log.Debugf("PUT %s", url)
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(data); err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, url, &buf)
	if err != nil {
		return err
	}
	req.Header = map[string][]string{
		"Content-Type": {"application/json"},
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if !valid(res) {
		return doError(res)
	}
	log.Debugf("reponse from server: %s", res.Status)
	return json.NewDecoder(res.Body).Decode(out)
}

func (c Client) post(url string, data, out interface{}) error {
	log.Debugf("POST %s: %v", url, data)
	buf := &bytes.Buffer{}
	if data != nil {
		if err := json.NewEncoder(buf).Encode(data); err != nil {
			return err
		}
	}
	return c.doPost(url, "application/json", buf, out)
}

func (c Client) doPost(url, ct string, r io.Reader, out interface{}) error {
	log.Debugf("POST %s", url)
	res, err := c.client.Post(url, ct, r)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if !valid(res) {
		return doError(res)
	}
	log.Debugf("reponse from server: %s", res.Status)
	// Requests that do not expect any data can set out = nil.
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func doError(res *http.Response) error {
	var ex Error
	if err := json.NewDecoder(res.Body).Decode(&ex); err != nil {
		return fmt.Errorf("bad response: %s", res.Status)
	}
	return fmt.Errorf("bad response: %d %s: %s", ex.Code, ex.Status, ex.Message)
}

func valid(res *http.Response) bool {
	return res.StatusCode >= http.StatusOK &&
		res.StatusCode < http.StatusMultipleChoices
}

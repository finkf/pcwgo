package api

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
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

// Client implements the api calls for the pcw backend.
// Use Login to initalize the client.
type Client struct {
	client  *http.Client
	Host    string
	Session Session // active session
}

// NewClient creates a new client with the given host (and it default
// web host).
func NewClient(host string, skipVerify bool) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify},
	}
	return &Client{
		Host:   host,
		client: &http.Client{Transport: tr},
	}
}

// Authenticate creates a new Client from a given auth-token.
func Authenticate(host, authToken string, skipVerify bool) *Client {
	c := NewClient(host, skipVerify)
	c.Session.Auth = authToken
	return c
}

// Login creates a new Client and authenticates with the given
// username and password.
func Login(host, email, password string, skipVerify bool) (*Client, error) {
	client := NewClient(host, skipVerify)
	login := LoginRequest{
		Email:    email,
		Password: password,
	}
	var s Session
	err := client.post(client.url("/login"), login, &s)
	if err != nil {
		return nil, err
	}
	log.Debugf("session: %s", s)
	client.Session = s
	return client, nil
}

// GetLogin returns the session of the authentificated user.
func (c Client) GetLogin() (Session, error) {
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

// DeleteUser deletes the user with the given id.
func (c Client) DeleteUser(id int64) error {
	return c.delete(c.url(userPath(id), Auth, c.Session.Auth))
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

// PostBook uploads a zipped OCR project and the given metadata.  It
// returns the newly created book.
func (c Client) PostBook(zip io.Reader, book Book) (*Book, error) {
	url := c.url("/books",
		Auth, c.Session.Auth,
		"author", book.Author,
		"title", book.Title,
		"language", book.Language,
		"description", book.Description,
		"histPatterns", book.HistPatterns,
		"profilerUrl", book.ProfilerURL,
		"year", strconv.Itoa(book.Year),
	)
	var newBook Book
	err := c.doPost(url, "application/zip", zip, &newBook)
	return &newBook, err
}

// PutBook updates the given book's metadata. It returns the updated
// book data.
func (c Client) PutBook(book Book) (*Book, error) {
	url := c.url(bookPath(book.ProjectID), Auth, c.Session.Auth)
	var newBook Book
	err := c.put(url, book, &newBook)
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

// DeleteBook deletes the book the given book.
func (c Client) DeleteBook(bookID int) error {
	url := c.url(bookPath(bookID), Auth, c.Session.Auth)
	return c.delete(url)
}

// GetPage returns the page with the given ids.
func (c Client) GetPage(bookID, pageID int) (*Page, error) {
	var page Page
	url := c.url(pagePath(bookID, pageID), Auth, c.Session.Auth)
	err := c.get(url, &page)
	return &page, err
}

// GetFirstPage returns the first page of the given book.
func (c Client) GetFirstPage(bookID int) (*Page, error) {
	var page Page
	url := c.url(bookPath(bookID)+"/pages/first", Auth, c.Session.Auth)
	err := c.get(url, &page)
	return &page, err
}

// GetLastPage returns the last page of the given book.
func (c Client) GetLastPage(bookID int) (*Page, error) {
	var page Page
	url := c.url(bookPath(bookID)+"/pages/last", Auth, c.Session.Auth)
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

// PutLine corrects is used to correct a line.
func (c Client) PutLine(bookID, pageID, lineID int, cor string) (*Line, error) {
	url := c.url(linePath(bookID, pageID, lineID), Auth, c.Session.Auth)
	post := struct {
		Correction string `json:"correction"`
	}{cor}
	var line Line
	err := c.put(url, post, &line)
	return &line, err
}

// DeleteLine deletes the given line, page or .
func (c Client) DeleteLine(bookID, pageID, lineID int) error {
	url := c.url(linePath(bookID, pageID, lineID), Auth, c.Session.Auth)
	return c.delete(url)
}

// GetToken returns the token for the given line.
func (c Client) GetToken(bookID, pageID, lineID, tokenID int) (*Token, error) {
	var token Token
	url := c.url(tokenPath(bookID, pageID, lineID, tokenID), Auth, c.Session.Auth)
	err := c.get(url, &token)
	return &token, err
}

// GetTokenLen returns the token with the given length.
func (c Client) GetTokenLen(bookID, pageID, lineID, offset, len int) (*Token, error) {
	var token Token
	url := c.url(tokenPath(bookID, pageID, lineID, offset),
		Auth, c.Session.Auth, "len", strconv.Itoa(len))
	err := c.get(url, &token)
	return &token, err
}

// PutToken corrects a token.
func (c Client) PutToken(bookID, pageID, lineID, tokenID int, cor string) (*Token, error) {
	url := c.url(tokenPath(bookID, pageID, lineID, tokenID), Auth, c.Session.Auth)
	post := struct {
		Correction string `json:"correction"`
	}{cor}
	var token Token
	err := c.put(url, post, &token)
	return &token, err
}

// PutTokenLen corrects a token of a spcific length.
func (c Client) PutTokenLen(bookID, pageID, lineID, tokenID, len int, cor string) (*Token, error) {
	url := c.url(tokenPath(bookID, pageID, lineID, tokenID),
		Auth, c.Session.Auth, "len", strconv.Itoa(len))
	post := struct {
		Correction string `json:"correction"`
	}{cor}
	var token Token
	err := c.put(url, post, &token)
	return &token, err
}

// SearchType defines the type of searches
type SearchType string

// Search types.
const (
	SearchToken   SearchType = "token"
	SearchPattern SearchType = "pattern"
	SearchAC      SearchType = "ac"
)

// Search is used configure and execute searches.
type Search struct {
	Client    Client     // API client used for the search
	Skip, Max int        // skip matches and max matches
	Type      SearchType // type of the search (if empty Type = token)
	IC        bool       // ignore case (applys only to Type = token)
}

// Search searches for the given queries.
func (s Search) Search(bookID int, qs ...string) (*SearchResults, error) {
	url := s.Client.url(bookPath(bookID)+"/search", s.params(s.Client.Session.Auth, qs...)...)
	var ret SearchResults
	err := s.Client.get(url, &ret)
	return &ret, err
}

func (s Search) params(auth string, qs ...string) []string {
	ret := []string{Auth, auth}
	ret = append(ret, "skip", strconv.Itoa(s.Skip))
	ret = append(ret, "max", strconv.Itoa(s.Max))
	if s.Type != "" {
		ret = append(ret, "t", string(s.Type))
	}
	for _, q := range qs {
		ret = append(ret, "q", q)
	}
	return ret
}

// GetAdaptiveTokens returns the adaptive tokens for the given book.
func (c Client) GetAdaptiveTokens(bookID int) (AdaptiveTokens, error) {
	url := c.url("/profile/adaptive/"+bookPath(bookID), Auth, c.Session.Auth)
	var res AdaptiveTokens
	err := c.get(url, &res)
	return res, err
}

// Split splits a project.
func (c Client) Split(pid int, random bool, uid int, ids ...int) (SplitPackages, error) {
	url := c.url("/pkg/split"+bookPath(pid), Auth, c.Session.Auth)
	post := SplitRequest{
		UserIDs: append(append([]int{}, uid), ids...),
		Random:  random,
	}
	var packages SplitPackages
	err := c.post(url, post, &packages)
	return packages, err
}

// AssignTo assigns a package to another user.  User must be an admin.
func (c Client) AssignTo(pid, uid int) error {
	url := c.url("/pkg/assign"+bookPath(pid),
		Auth, c.Session.Auth, "assignto", strconv.Itoa(uid))
	return c.get(url, nil)
}

// AssignBack assigns a package back to its original user. User must
// own the package.
func (c Client) AssignBack(pid int) error {
	url := c.url("/pkg/assign"+bookPath(pid), Auth, c.Session.Auth)
	return c.get(url, nil)
}

// TakeBack takes all packages of the given project back and reassigns
// them to the project's owner.  Only admins can take back projects.
func (c Client) TakeBack(pid int) error {
	url := c.url("/pkg/takeback"+bookPath(pid), Auth, c.Session.Auth)
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

// PostExtendedLexicon sends a request to create the extendedn
// lexicon for the given book or project.
func (c Client) PostExtendedLexicon(bookID int) (Job, error) {
	url := c.url("/postcorrect/le"+bookPath(bookID), Auth, c.Session.Auth)
	var job Job
	return job, c.post(url, nil, &job)
}

// GetExtendedLexicon returns the extended lexicon for the given book
// or project.
func (c Client) GetExtendedLexicon(bookID int) (ExtendedLexicon, error) {
	url := c.url("/postcorrect/le"+bookPath(bookID), Auth, c.Session.Auth)
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

// GetPostCorrection returns the post-correction data for the given book.
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

// OCRPredict runs the OCR prediction with the given model over the
// given book or project pages and/or lines.  If the give line and/or
// page ids are equal to zero the whole page and/or project are
// predicted.
func (c Client) OCRPredict(bid, pid, lid int, name string) (Job, error) {
	model := Model{Name: name}
	prefix := "/ocr/predict"
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

// OCRTrain trains a new model using the given model as base on the
// given project.
func (c Client) OCRTrain(bid int, name string) (Job, error) {
	model := Model{Name: name}
	url := c.url("/ocr/train"+bookPath(bid), Auth, c.Session.Auth)
	var job Job
	return job, c.post(url, model, &job)
}

// GetCharMap returns the frequency map of characters for the given
// book.
func (c Client) GetCharMap(bid int, filter string) (CharMap, error) {
	url := c.url(bookPath(bid)+"/charmap", Auth, c.Session.Auth, "filter", filter)
	var res CharMap
	return res, c.get(url, &res)
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

func (c Client) hostRoot() string {
	hostURL, err := url.Parse(c.Host)
	if err != nil {
		return c.Host
	}
	return strings.TrimSuffix(hostURL.String(), hostURL.RequestURI())
}

// GetLineImage downloads the line image for the given line.  At this
// point only PNGs are accepted.
func (c Client) GetLineImage(line *Line) (image.Image, error) {
	url := c.hostRoot() + "/" + line.ImgFile
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
	xurl = c.hostRoot() + "/" + archive.Archive
	log.Debugf("GET %s", xurl)
	res, err := c.client.Get(xurl)
	if err != nil {
		return nil, fmt.Errorf("cannot download: %v", err)
	}
	log.Debugf("GET %s: %s", xurl, res.Status)
	// do *not* defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("cannot download: bad response: %s", res.Status)
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
		if keyvals[i+1] == "" { // skip empty values
			continue
		}
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
	log.Debugf("GET %s: %s", url, res.Status)
	if !valid(res) {
		return doError(res)
	}
	if out == nil {
		return nil
	}
	return decodeJSONMaybeZipped(res, out)
}

func decodeJSONMaybeZipped(res *http.Response, out interface{}) error {
	var r io.Reader = res.Body
	if res.Header.Get("Content-Encoding") == "gzip" {
		log.Debugf("unzipping content")
		gzip, err := gzip.NewReader(r)
		if err != nil {
			return err
		}
		r = gzip
	}
	// log.Debugf("decodeJSON")
	// x, err := ioutil.ReadAll(r)
	// if err != nil {
	// 	return err
	// }
	// log.Debugf("decodeJSON")
	// log.Debugf("raw: %s", string(x))
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

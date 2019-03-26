package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/finkf/pcwgo/db"
	log "github.com/sirupsen/logrus"
)

// Client implements the api calls for the pcw backend.
// Use Login to initalize the client.
type Client struct {
	client  *http.Client
	Host    string
	Session Session
}

// Authenticate creates a new Client from a given auth-token.
func Authenticate(host, authToken string) *Client {
	return &Client{
		Host:    host,
		Session: Session{Auth: authToken},
		client:  &http.Client{},
	}
}

// Login creates a new Client and authenticates with the given
// username and password.
func Login(host, email, password string) (*Client, error) {
	// tr := &http.Transport{
	// 	TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify},
	// }
	c := &Client{
		client: &http.Client{}, //Transport: tr},
		Host:   host,
	}
	login := LoginRequest{
		Email:    email,
		Password: password,
	}
	var s Session
	err := c.post(c.url("/login"), login, &s)
	if err != nil {
		return nil, err
	}
	log.Debugf("session: %s", db.Session(s))
	c.Session = Session(s)
	return c, nil
}

func (c Client) getLogin() (Session, error) {
	var s Session
	err := c.get(c.url("/login", Auth, c.Session.Auth), &s)
	return s, err
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

// GetBooks() returns all books of a user.
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

// GetTokens returns the tokens for the given line.
func (c Client) GetTokens(bookID, pageID, lineID int) (Tokens, error) {
	var tokens Tokens
	url := c.url(linePath(bookID, pageID, lineID)+"/tokens", Auth, c.Session.Auth)
	err := c.get(url, &tokens)
	return tokens, err
}

// PostWord posts new content to the given token.
func (c Client) PostToken(bookID, pageID, lineID, tokenID int, cor Correction) (*Token, error) {
	var token Token
	url := c.url(tokenPath(bookID, pageID, lineID, tokenID), Auth, c.Session.Auth)
	err := c.post(url, cor, &token)
	return &token, err
}

// Search searches for tokens or error patterns.
func (c Client) Search(bookID int, query string, errorPattern bool) (*SearchResults, error) {
	p := "0"
	if errorPattern {
		p = "1"
	}
	url := c.url(bookPath(bookID)+"/search", "q", query, "p", p, Auth, c.Session.Auth)
	var res SearchResults
	err := c.get(url, &res)
	return &res, err
}

// PostProfile sends a request to profile the book with the given id.
func (c Client) PostProfile(bookID int) error {
	url := c.url(bookPath(bookID)+"/profile", Auth, c.Session.Auth)
	return c.post(url, nil, nil)
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
	if !IsValidJSONResponse(res, http.StatusOK) {
		return fmt.Errorf("bad response: %s [Content-Type: %s]",
			res.Status, res.Header.Get("Content-Type"))
	}
	log.Debugf("reponse from server: %s", res.Status)
	return json.NewDecoder(res.Body).Decode(out)
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
	if !IsValidJSONResponse(res, http.StatusOK) {
		return fmt.Errorf("bad response: %s [Content-Type: %s]",
			res.Status, res.Header.Get("Content-Type"))
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
	// Requests that do not expect any data can set out = nil.
	if out == nil {
		return nil
	}
	if !IsValidJSONResponse(res, http.StatusOK, http.StatusCreated) {
		return fmt.Errorf("bad response: %s [Content-Type: %s]",
			res.Status, res.Header.Get("Content-Type"))
	}
	log.Debugf("reponse from server: %s", res.Status)
	return json.NewDecoder(res.Body).Decode(out)
}

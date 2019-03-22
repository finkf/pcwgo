package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/finkf/pcwgo/database/session"
	"github.com/finkf/pcwgo/database/user"
	log "github.com/sirupsen/logrus"
)

// Client implements the api calls for the pcw backend.
// Use Login to initalize the client.
type Client struct {
	client  *http.Client
	Host    string
	Session session.Session
}

// Authenticate creates a new Client from a given auth-token.
func Authenticate(host, authToken string) *Client {
	return &Client{
		Host:    host,
		Session: session.Session{Auth: authToken},
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
	var s session.Session
	err := c.post(c.url("/login"), login, &s)
	if err != nil {
		return nil, err
	}
	log.Debugf("session: %s", s)
	c.Session = s
	return c, nil
}

func (c Client) getLogin() (session.Session, error) {
	var s session.Session
	err := c.get(c.url("/login", Auth, c.Session.Auth), &s)
	return s, err
}

// GetUsers returns the list of users.
func (c Client) GetUsers() ([]user.User, error) {
	var res []user.User
	err := c.get(c.url("/users", Auth, c.Session.Auth), &res)
	return res, err
}

// GetUser returns the user with the given id.
func (c Client) GetUser(id int64) (user.User, error) {
	var res user.User
	err := c.get(c.url(userPath(id), Auth, c.Session.Auth), &res)
	return res, err
}

// PutUser updates the settings for a user and returns it.
func (c Client) PutUser(u CreateUserRequest) (user.User, error) {
	var res user.User
	url := c.url(userPath(u.User.ID), Auth, c.Session.Auth)
	err := c.put(url, u, &res)
	return res, err
}

// PostUser creates a new User and returns it.
func (c Client) PostUser(u CreateUserRequest) (user.User, error) {
	var res user.User
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
func (c Client) PostZIP(zip io.Reader) (Book, error) {
	var book Book
	url := c.url("/books", Auth, c.Session.Auth)
	err := c.doPost(url, "application/zip", zip, &book)
	return book, err
}

// PostBook updates the given Book and returns it.
func (c Client) PostBook(book Book) (Book, error) {
	url := c.url(bookPath(book.ProjectID), Auth, c.Session.Auth)
	var newBook Book
	err := c.post(url, book, &newBook)
	return newBook, err
}

// GetPage returns the page with the given ids.
func (c Client) GetPage(bookID, pageID int) (Page, error) {
	var page Page
	url := c.url(pagePath(bookID, pageID), Auth, c.Session.Auth)
	err := c.get(url, &page)
	return page, err
}

// PostProfile sends a request to profile the book with the given id.
func (c Client) PostProfile(bookID int) error {
	url := c.url(bookPath(bookID)+"/profile", Auth, c.Session.Auth)
	return c.post(url, nil, nil)
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
	return formatID("/books/%d/pages", id, pageid)
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

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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
	err := c.post(fmt.Sprintf("%s/login", host), login, &s)
	if err != nil {
		return nil, err
	}
	log.Debugf("session: %s", s)
	c.Session = s
	return c, nil
}

func (c Client) getLogin() (session.Session, error) {
	var s session.Session
	err := c.get(fmt.Sprintf("%s/login?%s=%s",
		c.Host, Auth, c.Session.Auth), &s)
	return s, err
}

func (c Client) getUsers() ([]user.User, error) {
	var res []user.User
	err := c.get(fmt.Sprintf("%s/users?auth=%s",
		c.Host, c.Session.Auth), &res)
	return res, err
}

func (c Client) getUser(id int64) (user.User, error) {
	var res user.User
	err := c.get(fmt.Sprintf("%s/users/%d?auth=%s",
		c.Host, id, c.Session.Auth), &res)
	return res, err
}

func (c Client) putUser(u CreateUserRequest) (user.User, error) {
	var res user.User
	err := c.put(fmt.Sprintf("%s/users/%d?auth=%s",
		c.Host, u.User.ID, c.Session.Auth), u, &res)
	return res, err
}

func (c Client) postUser(u CreateUserRequest) (user.User, error) {
	var res user.User
	err := c.post(fmt.Sprintf("%s/create-user?auth=%s",
		c.Host, c.Session.Auth), u, &res)
	return res, err
}

func (c Client) getAPIVersion() (Version, error) {
	var res Version
	err := c.get(fmt.Sprintf("%s/api-version", c.Host), &res)
	return res, err
}

func (c Client) postZIP(zip io.Reader) (Book, error) {
	var book Book
	url := fmt.Sprintf("%s/books?auth=%s", c.Host, c.Session.Auth)
	if err := c.doPost(url, "application/zip", zip, &book); err != nil {
		return Book{}, err
	}
	return book, nil
}

func (c Client) postBook(book Book) error {
	url := fmt.Sprintf("%s/books/%d?auth=%s",
		c.Host, book.ProjectID, c.Session.Auth)
	return c.post(url, book, nil)
}

func (c Client) getPage(bookID, pageID int) (Page, error) {
	var page Page
	url := fmt.Sprintf("%s/books/%d/pages/%d?auth=%s",
		c.Host, bookID, pageID, c.Session.Auth)
	if err := c.get(url, &page); err != nil {
		return Page{}, err
	}
	return page, nil
}

func (c Client) postProfile(bookID int) error {
	url := fmt.Sprintf("%s/books/%d/profile?auth=%s",
		c.Host, bookID, c.Session.Auth)
	return c.post(url, nil, nil)
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

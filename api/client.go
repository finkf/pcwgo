package api

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/UNO-SOFT/ulog"
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
	err := client.Post(client.URL("login"), login, &s)
	if err != nil {
		return nil, err
	}
	client.Session = s
	return client, nil
}

// URL returns the formated url with the client's host prepended.
func (c Client) URL(format string, args ...interface{}) string {
	return strings.TrimRight(c.Host, "/") + "/" + strings.TrimLeft(fmt.Sprintf(format, args...), "/")
}

// Do performes an authenticated HTTP request against a pocoweb
// service.
func (c Client) Do(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", c.Session.Auth)
	return c.client.Do(req)
}

// Get performes an authenticated HTTP get request against a pocoweb
// service.  The response of the request is marshaled into the out
// parameter unless the out parameter is set to nil.
func (c Client) Get(url string, out interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("GET %s: %v", url, err)
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %v", url, err)
	}
	if err := UnmarshalResponse(resp, out); err != nil {
		return fmt.Errorf("GET %s: %v", url, err)
	}
	return nil
}

// Post performes an authenticated HTTP post request against a pocoweb
// service with the given payload formatted as json.  The response of
// the request is marshaled into the out parameter unless the out
// parameter is set to nil.
func (c Client) Post(url string, payload, out interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("POST %s: %v", url, err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("POST %s: %v", url, err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %v", url, err)
	}
	if err := UnmarshalResponse(resp, out); err != nil {
		return fmt.Errorf("POST %s: %v", url, err)
	}
	return nil
}

// Put performes an authenticated HTTP put request against a pocoweb
// service with the given payload formatted as json.  The response of
// the request is marshaled into the out parameter unless the out
// parameter is set to nil.
func (c Client) Put(url string, payload, out interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("PUT %s: %v", url, err)
	}
	ulog.Write("Put", "payload", string(body))
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("PUT %s: %v", url, err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("PUT %s: %v", url, err)
	}
	if err := UnmarshalResponse(resp, out); err != nil {
		return fmt.Errorf("PUT %s: %v", url, err)
	}
	return nil
}

// Delete performes an authenticated HTTP delete request against a
// pocoweb service with the given payload formatted as json.  The
// response of the request is marshaled into the out parameter unless
// the out parameter is set to nil.
func (c Client) Delete(url string, out interface{}) error {
	req, err := http.NewRequest(http.MethodDelete, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("DELETE %s: %v", url, err)
	}
	req.Header.Add("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("DELETE %s: %v", url, err)
	}
	if err := UnmarshalResponse(resp, out); err != nil {
		return fmt.Errorf("DELETE %s: %v", url, err)
	}
	return nil
}

// UnmarshalResponse unmarshals the response of a pocoweb api into to
// the given output parameter.  The content of the response is assumed
// to be (gzipped) json-encoded.  The response body is closed and
// possible api errors are handled.  If out is set to nil, the
// response data (if any) is discarded.
func UnmarshalResponse(resp *http.Response, out interface{}) error {
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		errresp := ErrorResponse{
			Status:     resp.Status,
			StatusCode: resp.StatusCode,
		}
		err := json.Unmarshal(body, &errresp)
		if err != nil {
			return errresp
		}
		return errresp
	}
	if out == nil {
		return nil
	}
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzip, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return err
		}
		return json.NewDecoder(gzip).Decode(out)
	}
	return json.Unmarshal(body, out)
}

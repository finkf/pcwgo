package api

import (
	"fmt"
	"net/http"

	"github.com/finkf/pcwgo/database/user"
)

const (
	// Auth definest the ?auth=xxx token
	Auth = "auth"
)

// LoginRequest defines the login data.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// String returns the string representation of
// a login request. The Password is *not* printed.
func (l LoginRequest) String() string {
	return "{" + l.Email + " ***}"
}

// CreateUserRequest defines the data to create new users.
type CreateUserRequest struct {
	User     user.User `json:"user"`
	Password string    `json:"password"`
}

// ErrorResponse defines the data of error responses
type ErrorResponse struct {
	Cause      string `json:"cause"`
	Status     string `json:"status"`
	StatusCode int    `json:"statusCode"`
}

// Version defines the resonse data of version requests.
type Version struct {
	Version string `json:"version"`
}

// Users defines the repsonse data for requests to list the system's users.
type Users struct {
	Users []User `json:"users"`
}

// NewErrorResponse creates a new ErrorResponse with
// the given code and cause. The status text is calculated
// automatically using http.StatusText.
func NewErrorResponse(code int, cause string) ErrorResponse {
	return ErrorResponse{
		StatusCode: code,
		Cause:      cause,
		Status:     http.StatusText(code),
	}
}

func (err ErrorResponse) Error() string {
	if err.Cause == "" {
		return fmt.Sprintf("%d %s", err.StatusCode, err.Status)
	}
	return fmt.Sprintf("%s [%d %s]", err.Cause, err.StatusCode, err.Status)
}

// IsValidJSONResponse returns true if the given response matches
// on of the given codes and if the Content-Type equals the given
// content type.
func IsValidJSONResponse(res *http.Response, codes ...int) bool {
	// Order matters here. Check first for the return codes.
	var codeOK bool
	for _, code := range codes {
		if res.StatusCode == code {
			codeOK = true
			break
		}
	}
	if !codeOK {
		return false
	}
	// Then check for a matching content type.
	for _, ct := range res.Header["Content-Type"] {
		if ct == "application/json" {
			return true
		}
	}
	return false
}

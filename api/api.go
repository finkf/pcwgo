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
	Users []user.User `json:"users"`
}

// Book defines the response data for books.
type Book struct {
	Author      string `json:"author"`
	Title       string `json:"title"`
	Language    string `json:"language"`
	ProfilerURL string `json:"profilerUrl"`
	Description string `json:"description"`
	Year        int    `json:"year"`
	ProjectID   int    `json:"projectId"`
	Pages       int    `json:"pages"`
	PageIDs     []int  `json:"pageIds"`
	IsBook      bool   `json:"isBook"`
}

// Books defines a list of books.
type Books struct {
	Books []Book `json:"books"`
}

// Page defines a page in a book.
type Page struct {
	PageID    int    `json:"pageId"`
	ProjectID int    `json:"projectId"`
	OCRFile   string `json:"ocrFile"`
	ImgFile   string `json:"imgFile"`
	Box       Box    `json:"box"`
	Lines     []Line `json:"lines"`
}

// Line defines the line of a page in a book.
type Line struct {
	ImgFile              string    `json:"imgFile"`
	Cor                  string    `json:"cor"`
	OCR                  string    `json:"ocr"`
	LineID               int       `json:"lineId"`
	PageID               int       `json:"pageId"`
	ProjectID            int       `json:"projectId"`
	Cuts                 []int     `json:"cuts"`
	Confidences          []float64 `json:"confidences"`
	AverageConfidence    float64   `json:"averageConfidence"`
	IsFullyCorrected     bool      `json:"isFullyCorrected"`
	IsPartiallyCorrected bool      `json:"isPartiallyCorrected"`
	Box                  Box       `json:"box"`
}

// Box defines the bounding box in an image.
type Box struct {
	Left   int `json:"left"`
	Right  int `json:"right"`
	Top    int `json:"top"`
	Bottom int `json:"bottom"`
	Width  int `json:"width"`
	Height int `json:"heigth"`
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

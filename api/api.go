package api

import (
	"fmt"
	"net/http"
	"time"
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
	User     User   `json:"user"`
	Password string `json:"password"`
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

// Session defines an authenticates user sessions.  A session is
// attached to a unique user with and authentication string and an
// expiration date.
type Session struct {
	User    User   `json:"user"`
	Auth    string `json:"auth"`
	Expires int64  `json:"expires"`
}

// Expired returns true if the session has exprired.
func (s Session) Expired() bool {
	if s.Expires < time.Now().Unix() {
		return true
	}
	return false
}

func (s Session) String() string {
	return fmt.Sprintf("%s [%s] expires: %s",
		s.User, s.Auth, time.Unix(s.Expires, 0).Format("2006-01-02:15:04"))
}

// SplitRequest defines the post data for split requests.
type SplitRequest struct {
	UserIDs []int `json:"userIds"`
	Random  bool  `json:"random"`
}

// SplitPackages defines the response data of split requests.
type SplitPackages struct {
	BookID   int            `json:"bookId"`
	Packages []SplitPackage `json:"projects"`
}

// SplitPackage defines the data for one split package.
type SplitPackage struct {
	PageIDs   []int `json:"pageIds"`
	ProjectID int   `json:"projectId"`
	Owner     int   `json:"owner"`
}

// User defines basic users.
type User struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Institute string `json:"institute"`
	ID        int64  `json:"id"`
	Admin     bool   `json:"admin"`
}

func (u User) String() string {
	adm := ""
	if u.Admin {
		adm = "*"
	}
	return fmt.Sprintf("%s(%d%s)", u.Email, u.ID, adm)
}

// Users defines the repsonse data for requests to list the system's users.
type Users struct {
	Users []User `json:"users"`
}

// BookWithPages is a Book with an additional field that holds all the
// book's pages.
type BookWithPages struct {
	Book
	PageContent []Page
}

// Book defines the response data for books.
type Book struct {
	Author       string          `json:"author"`
	Title        string          `json:"title"`
	Language     string          `json:"language"`
	Status       map[string]bool `json:"status"`
	ProfilerURL  string          `json:"profilerUrl"`
	Description  string          `json:"description"`
	HistPatterns string          `json:"histPatterns"`
	Year         int             `json:"year"`
	BookID       int             `json:"bookId"`
	ProjectID    int             `json:"projectId"`
	Pages        int             `json:"pages"`
	PageIDs      []int           `json:"pageIds"`
	IsBook       bool            `json:"isBook"`
	Pooled       bool            `json:"pooled"`
}

// Books defines a list of books.
type Books struct {
	Books []Book `json:"books"`
}

// Page defines a page in a book.
type Page struct {
	PageID     int    `json:"pageId"`
	ProjectID  int    `json:"projectId"`
	BookID     int    `json:"bookId"`
	PrevPageID int    `json:"prevPageId"`
	NextPageID int    `json:"nextPageId"`
	OCRFile    string `json:"ocrFile"`
	ImgFile    string `json:"imgFile"`
	Box        Box    `json:"box"`
	Lines      []Line `json:"lines"`
}

// Line defines the line of a page in a book.
type Line struct {
	ImgFile              string    `json:"imgFile"`
	Cor                  string    `json:"cor"`
	OCR                  string    `json:"ocr"`
	LineID               int       `json:"lineId"`
	PageID               int       `json:"pageId"`
	ProjectID            int       `json:"projectId"`
	BookID               int       `json:"bookId"`
	Cuts                 []int     `json:"cuts"`
	Confidences          []float64 `json:"confidences"`
	AverageConfidence    float64   `json:"averageConfidence"`
	IsFullyCorrected     bool      `json:"isFullyCorrected"`
	IsPartiallyCorrected bool      `json:"isPartiallyCorrected"`
	Box                  Box       `json:"box"`
	Tokens               []Token   `json:"tokens"`
}

// Token defines a token on a line.
type Token struct {
	Cor                  string    `json:"cor"`
	OCR                  string    `json:"ocr"`
	TokenID              int       `json:"tokenId"`
	LineID               int       `json:"lineId"`
	PageID               int       `json:"pageId"`
	ProjectID            int       `json:"projectId"`
	BookID               int       `json:"bookId"`
	Offset               int       `json:"offset"`
	Cuts                 []int     `json:"cuts"`
	Confidences          []float64 `json:"confidences"`
	AverageConfidence    float64   `json:"averageConfidence"`
	IsFullyCorrected     bool      `json:"isFullyCorrected"`
	IsPartiallyCorrected bool      `json:"isPartiallyCorrected"`
	IsNormal             bool      `json:"isNormal"`
	IsMatch              bool      `json:"match"`
	Box                  Box       `json:"box"`
}

// CharMap represents a freqency list of characters.
type CharMap struct {
	ProjectID int            `json:"projectId"`
	BookID    int            `json:"bookId"`
	CharMap   map[string]int `json:"charMap"`
}

// Tokens defines the tokens on a line.
type Tokens struct {
	Tokens []Token `json:"tokens"`
}

// Box defines the bounding box in an image.
type Box struct {
	Left   int `json:"left"`
	Right  int `json:"right"`
	Top    int `json:"top"`
	Bottom int `json:"bottom"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// SearchResults defines the results for token searches.
type SearchResults struct {
	Matches        map[string]Match `json:"matches"`
	BookID         int              `json:"bookId"`
	ProjectID      int              `json:"projectId"`
	Total          int              `json:"total"`
	Max            int              `json:"max"`
	Skip           int              `json:"skip"`
	IsErrorPattern bool             `json:"isErrorPattern"`
}

// CorrectionRequest defines the payload for correction request for
// lines and/or tokens.
type CorrectionRequest struct {
	Correction string `json:"correction"`
	Manually   bool   `json:"manually"`
}

// Match defines the matches in the results of searches.
type Match struct {
	Lines []Line `json:"lines"`
	Total int    `json:"total"`
}

// Suggestions defines the profiler's suggestions for tokens.
type Suggestions struct {
	Suggestions map[string][]Suggestion `json:"suggestions"`
	BookID      int                     `json:"bookId"`
	ProjectID   int                     `json:"projectId"`
}

// SuggestionCounts holds the counts of correction suggestions.
type SuggestionCounts struct {
	Counts    map[string]int `json:"counts"`
	BookID    int            `json:"bookId"`
	ProjectID int            `json:"projectId"`
}

// Suggestion defines one suggestion of the profiler for a token.
type Suggestion struct {
	Token        string   `json:"token"`
	Suggestion   string   `json:"suggestion"`
	Modern       string   `json:"modern"`
	Dict         string   `json:"dict"`
	Distance     int      `json:"distance"`
	ID           int      `json:"id"`
	Weight       float64  `json:"weight"`
	Top          bool     `json:"top"`
	OCRPatterns  []string `json:"ocrPatterns"`
	HistPatterns []string `json:"histPatterns"`
}

// PatternCounts holds the pattern counts for error patterns.
type PatternCounts struct {
	Counts    map[string]int `json:"counts"`
	BookID    int            `json:"bookId"`
	ProjectID int            `json:"projectId"`
	OCR       bool           `json:"ocr"`
}

// Patterns holds patterns.
type Patterns struct {
	Patterns  map[string][]Suggestion `json:"patterns"`
	BookID    int                     `json:"bookId"`
	ProjectID int                     `json:"projectId"`
	OCR       bool                    `json:"ocr"`
}

// AdaptiveTokens holds a list of adaptive tokens.
type AdaptiveTokens struct {
	BookID         int      `json:"bookId"`
	ProjectID      int      `json:"projectId"`
	AdaptiveTokens []string `json:"adaptiveTokens"`
}

// ExtendedLexicon defines the object returned as the result for the
// lexicon extension.
type ExtendedLexicon struct {
	BookID    int            `json:"bookId"`
	ProjectID int            `json:"projectId"`
	Yes       map[string]int `json:"yes"`
	No        map[string]int `json:"no"`
}

// AdditionalLexicon represents the additional lexicon tokens for th
// post data of the postcorrection and the profiler.
type AdditionalLexicon struct {
	Tokens []string `json:"tokens"`
}

// PostCorrection represents the result of the post correction.  It
// maps tokens with unique id strings (bookID:pageID:lineID:tokenID)
// to correction decisions of the automatical post correction.
type PostCorrection struct {
	BookID      int                            `json:"bookId"`
	ProjectID   int                            `json:"projectId"`
	Corrections map[string]PostCorrectionToken `json:"corrections"`
}

// PostCorrectionToken represent unique post corrected tokens.
type PostCorrectionToken struct {
	BookID     int     `json:"bookId"`
	ProjectID  int     `json:"projectId"`
	PageID     int     `json:"pageId"`
	LineID     int     `json:"lineId"`
	TokenID    int     `json:"tokenId"`
	Normalized string  `json:"normalized"`
	OCR        string  `json:"ocr"`
	Cor        string  `json:"cor"`
	Confidence float64 `json:"confidence"`
	Taken      bool    `json:"taken"`
}

// Job defines the job struct.
type Job struct {
	ID int `json:"id"`
}

// JobStatus defines the job status struct.
type JobStatus struct {
	JobID      int    `json:"jobId"`
	BookID     int    `json:"bookId"`
	StatusID   int    `json:"statusId"`
	StatusName string `json:"statusName"`
	JobName    string `json:"jobName"`
	Timestamp  int64  `json:"timestamp"`
}

// Time returns the time object for the job's timestamp.
func (js JobStatus) Time() time.Time {
	return time.Unix(js.Timestamp, 0)
}

// Languages defines the object that contains the profiler's
// configured languages.
type Languages struct {
	Languages []string `json:"languages"`
}

// Model defines the ocr models.
type Model struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Models defines multiple models.
type Models struct {
	Models []Model `json:"models"`
}

// Error defines json-formatted error responses.
type Error struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
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

// IsValidJSONResponse returns true if the given response matches one
// of the given codes and if response is either empty or its
// Content-Type is `application/json`.
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
	// Then check if we do have any content.
	if res.ContentLength == 0 {
		return true
	}
	// Finally check for a matching content type.
	for _, ct := range res.Header["Content-Type"] {
		if ct == "application/json" {
			return true
		}
	}
	return false
}

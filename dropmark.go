package dropmark

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProgressReporter is sent to this package's methods if activity progress reporting is expected
type ProgressReporter interface {
	StartReportableReaderActivityInBytes(summary string, exepectedBytes int64, inputReader io.Reader) io.Reader
	CompleteReportableActivityProgress(summary string)
}

// HTTPUserAgent may be passed into GetDropmarkCollection as the default HTTP User-Agent header parameter
const HTTPUserAgent = "github.com/lectio/dropmark"

// HTTPTimeout may be passed into the GetDropmarkCollection as the default timeout parameter
const HTTPTimeout = time.Second * 90

// Collection is the object returned from the Dropmark API calls after JSON unmarshalling is completed
type Collection struct {
	Name  string  `json:"name,omitempty"`
	Items []*Item `json:"items,omitempty"`

	apiEndpoint string
	issues      []Issue `json:"issues"`
}

// Content satisfies the general Lectio interface for retrieving a single piece of content from a list
func (c Collection) Content() (count int, itemFn func(startIndex, endIndex int) (interface{}, error), err error) {
	count = len(c.Items)
	itemFn = func(startIndex, endIndex int) (interface{}, error) {
		if startIndex == endIndex {
			return c.Items[startIndex], nil
		}
		list := make([]*Item, endIndex-startIndex+1)
		listIndex := 0
		for i := startIndex; i <= endIndex; i++ {
			list[listIndex] = c.Items[i]
			listIndex++
		}
		return list, nil
	}
	return
}

// Source returns the Dropmark API endpoint which created the collection
func (c Collection) Source() string {
	return c.apiEndpoint
}

// ErrorsAndWarnings contains the problems in this link plus satisfies the Link.Issues interface
func (c Collection) ErrorsAndWarnings() []Issue {
	return c.issues
}

// IssueCounts returns the total, errors, and warnings counts
func (c Collection) IssueCounts() (uint, uint, uint) {
	if c.issues == nil {
		return 0, 0, 0
	}
	var errors, warnings uint
	for _, i := range c.issues {
		if i.IsError() {
			errors++
		} else {
			warnings++
		}
	}
	return uint(len(c.issues)), errors, warnings
}

// HandleIssues loops through each issue and calls a particular handler
func (c Collection) HandleIssues(errorHandler func(Issue), warningHandler func(Issue)) {
	if c.issues == nil {
		return
	}
	for _, i := range c.issues {
		if i.IsError() && errorHandler != nil {
			errorHandler(i)
		}
		if i.IsWarning() && warningHandler != nil {
			warningHandler(i)
		}
	}
}

// Tidy cleans up some of the problems in the source items
func (c *Collection) tidy() {
	for i, item := range c.Items {
		item.tidy(i)
	}
}

// Thumbnails represents a group of images
type Thumbnails struct {
	Mini      string `json:"mini,omitempty"`
	Small     string `json:"small,omitempty"`
	Large     string `json:"large,omitempty"`
	Cropped   string `json:"cropped,omitempty"`
	Uncropped string `json:"uncropped,omitempty"`
}

// Tag represents a single tag
type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
}

// Item represents a single Dropmark collection item after JSON unmarshalling is completed
type Item struct {
	Link            string      `json:"link,omitempty"`
	Name            string      `json:"name,omitempty"`
	Description     string      `json:"description,omitempty"`
	Content         string      `json:"content,omitempty"`
	Tags            []*Tag      `json:"tags,omitempty"`
	CreatedAt       string      `json:"created_at,omitempty"`
	UpdatedAt       string      `json:"updated_at,omitempty"`
	ThumbnailURL    string      `json:"thumbnail,omitempty"`
	Thumbnails      *Thumbnails `json:"thumbnails,omitempty"`
	UserID          string      `json:"user_id,omitempty"`
	UserNameShort   string      `json:"username,omitempty"`
	UserNameLong    string      `json:"user_name,omitempty"`
	UserEmail       string      `json:"user_email,omitempty"`
	UserAvatarURL   *Thumbnails `json:"user_avatar,omitempty"`
	DropmarkEditURL string      `json:"url"`

	edits []string
}

// TargetURL satisfies the contract for a Lectio Link object
func (i Item) TargetURL() string {
	return i.Link
}

func (i *Item) tidy(index int) {
	_, contentURLErr := url.Parse(i.Content)
	if contentURLErr == nil {
		// Sometimes in Dropmark, the content is just a URL (not sure why).
		// If the entire content is just a single URL, replace it with the Description
		i.edits = append(i.edits, fmt.Sprintf("Item[%d].Content was a URL %q, replaced with Description", index, i.Content))
		i.Content = i.Description
	}

	if strings.Compare(i.Content, i.Description) == 0 {
		i.edits = append(i.edits, fmt.Sprintf("Item[%d].Content was the same as the Description, set Description to blank", index))
		i.Description = ""
	}
}

// GetCollection takes a Dropmark apiEndpoint and creates a Collection object
func GetCollection(apiEndpoint string, pr ProgressReporter, userAgent string, timeout time.Duration) (*Collection, Issues) {
	result := new(Collection)
	result.apiEndpoint = apiEndpoint

	httpClient := http.Client{
		Timeout: timeout,
	}
	req, reqErr := http.NewRequest(http.MethodGet, apiEndpoint, nil)
	if reqErr != nil {
		result.issues = append(result.issues, newIssue(apiEndpoint, UnableToCreateHTTPRequest, fmt.Sprintf("Unable to create HTTP request: %v", reqErr), true))
		return nil, result
	}
	req.Header.Set("User-Agent", userAgent)
	resp, getErr := httpClient.Do(req)
	if getErr != nil {
		result.issues = append(result.issues, newIssue(apiEndpoint, UnableToExecuteHTTPGETRequest, fmt.Sprintf("Unable to execute HTTP GET request: %v", getErr), true))
		return nil, result
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.issues = append(result.issues, newIssue(apiEndpoint, InvalidAPIRespHTTPStatusCode, fmt.Sprintf("Dropmark API status is not HTTP OK (200): %v", resp.StatusCode), true))
		return nil, result
	}

	var body []byte
	var readErr error
	reader := pr.StartReportableReaderActivityInBytes(fmt.Sprintf("Processing Dropmark API request %q (%d bytes)", apiEndpoint, resp.ContentLength), resp.ContentLength, resp.Body)
	body, readErr = ioutil.ReadAll(reader)

	if readErr != nil {
		result.issues = append(result.issues, newIssue(apiEndpoint, UnableToReadBodyFromHTTPResponse, fmt.Sprintf("Unable to read body from HTTP response: %v", readErr), true))
		return nil, result
	}

	json.Unmarshal(body, result)
	result.tidy()

	pr.CompleteReportableActivityProgress(fmt.Sprintf("Completed Dropmark API request %q with %d items", apiEndpoint, len(result.Items)))

	return result, nil
}

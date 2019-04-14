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
	IsProgressReportingRequested() bool
	StartReportableReaderActivityInBytes(exepectedBytes int64, inputReader io.Reader) io.Reader
	CompleteReportableActivityProgress(summary string)
}

// HTTPUserAgent may be passed into GetDropmarkCollection as the default HTTP User-Agent header parameter
const HTTPUserAgent = "github.com/lectio/dropmark"

// HTTPTimeout may be passed into the GetDropmarkCollection as the default timeout parameter
const HTTPTimeout = time.Second * 90

// Collection is the object returned from the Dropmark API calls after JSON unmarshalling is completed
type Collection struct {
	Name        string  `json:"name,omitempty"`
	Items       []*Item `json:"items,omitempty"`
	apiEndpoint string
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
func GetCollection(apiEndpoint string, pr ProgressReporter, userAgent string, timeout time.Duration) (*Collection, error) {
	result := new(Collection)
	result.apiEndpoint = apiEndpoint

	httpClient := http.Client{
		Timeout: timeout,
	}
	req, reqErr := http.NewRequest(http.MethodGet, apiEndpoint, nil)
	if reqErr != nil {
		return nil, fmt.Errorf("Unable to create request %q: %v", apiEndpoint, reqErr)
	}
	req.Header.Set("User-Agent", userAgent)
	resp, getErr := httpClient.Do(req)
	if getErr != nil {
		return nil, fmt.Errorf("Unable to execute GET request %q: %v", apiEndpoint, getErr)
	}
	defer resp.Body.Close()

	var body []byte
	var readErr error
	if pr != nil && pr.IsProgressReportingRequested() {
		reader := pr.StartReportableReaderActivityInBytes(resp.ContentLength, resp.Body)
		body, readErr = ioutil.ReadAll(reader)
	} else {
		body, readErr = ioutil.ReadAll(resp.Body)
	}

	if readErr != nil {
		return nil, fmt.Errorf("Unable to read body from request %q: %v", apiEndpoint, readErr)
	}

	json.Unmarshal(body, result)
	result.tidy()

	if pr != nil && pr.IsProgressReportingRequested() {
		pr.CompleteReportableActivityProgress(fmt.Sprintf("Completed Dropmark API request %q with %d items", apiEndpoint, len(result.Items)))
	}

	return result, nil
}

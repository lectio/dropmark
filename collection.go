package dropmark

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Collection is the object returned from the Dropmark API calls after JSON unmarshalling is completed
type Collection struct {
	Name  string  `json:"name,omitempty"`
	Items []*Item `json:"items,omitempty"`

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

// ContentSourceName returns name of the Dropmark API source
func (c Collection) ContentSourceName() string {
	return "Dropmark"
}

// ContentAPIEndpoint returns the Dropmark API endpoint which created the collection
func (c Collection) ContentAPIEndpoint() string {
	return c.apiEndpoint
}

// Tidy cleans up some of the problems in the source items
func (c *Collection) tidy() {
	for i, item := range c.Items {
		item.tidy(i)
	}
}

// GetCollection takes a Dropmark apiEndpoint and creates a Collection object
func GetCollection(ctx context.Context, apiEndpoint string, options ...interface{}) (*Collection, error) {
	result := new(Collection)
	result.apiEndpoint = apiEndpoint

	var client *http.Client
	var pr ReaderProgressReporter = defaultProgressReporter

	type prepareHTTPRequest interface {
		OnPrepareHTTPRequest(ctx context.Context, client *http.Client, req *http.Request)
	}
	var preReqOption prepareHTTPRequest

	for _, option := range options {
		if v, ok := option.(interface{ HTTPClient() *http.Client }); ok {
			client = v.HTTPClient()
		}
		if v, ok := option.(prepareHTTPRequest); ok {
			preReqOption = v
		}
		if v, ok := option.(ReaderProgressReporter); ok {
			pr = v
		}
	}

	if client == nil {
		client = &http.Client{
			Timeout: time.Second * 90,
		}
	}

	req, reqErr := http.NewRequest(http.MethodGet, apiEndpoint, nil)
	if reqErr != nil {
		return nil, fmt.Errorf("Unable to create HTTP request: %v", reqErr)
	}

	if preReqOption != nil {
		preReqOption.OnPrepareHTTPRequest(ctx, client, req)
	}

	resp, getErr := client.Do(req)
	if getErr != nil {
		return nil, fmt.Errorf("Unable to execute HTTP GET request: %v", getErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Dropmark API status is not HTTP OK (200): %v", resp.StatusCode)
	}

	var body []byte
	var readErr error
	reader := pr.StartReportableReaderActivityInBytes(fmt.Sprintf("Processing Dropmark API request %q (%d bytes)", apiEndpoint, resp.ContentLength), resp.ContentLength, resp.Body)
	body, readErr = ioutil.ReadAll(reader)

	if readErr != nil {
		return nil, fmt.Errorf("Unable to read body from HTTP response: %v", readErr)
	}

	json.Unmarshal(body, result)
	result.tidy()

	pr.CompleteReportableActivityProgress(fmt.Sprintf("Completed Dropmark API request %q with %d items", apiEndpoint, len(result.Items)))

	return result, nil
}

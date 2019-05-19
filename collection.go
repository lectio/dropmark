package dropmark

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type httpRequestPreparer interface {
	OnPrepareHTTPRequest(ctx context.Context, client *http.Client, req *http.Request)
}

type tidyHandler interface {
	OnTidy(ctx context.Context, tidy string)
}

// Collection is the object returned from the Dropmark API calls after JSON unmarshalling is completed
type Collection struct {
	APIEndpoint string  `json:"apiEndpoint,omitempty"` // injected by Lectio
	Name        string  `json:"name,omitempty"`        // from Dropmark API
	Items       []*Item `json:"items,omitempty"`       // from Dropmark API

	client      *http.Client
	reqPreparer httpRequestPreparer
	prepReqFunc func(ctx context.Context, client *http.Client, req *http.Request)
	rpr         ReaderProgressReporter
	bpr         BoundedProgressReporter
	tidyHandler tidyHandler
}

func (c *Collection) initOptions(ctx context.Context, apiEndpoint string, options ...interface{}) {
	c.APIEndpoint = apiEndpoint
	c.rpr = defaultProgressReporter
	c.bpr = defaultProgressReporter

	for _, option := range options {
		if v, ok := option.(interface {
			HTTPClient(ctx context.Context) *http.Client
		}); ok {
			c.client = v.HTTPClient(ctx)
		}
		if v, ok := option.(func(ctx context.Context) *http.Client); ok {
			c.client = v(ctx)
		}
		if v, ok := option.(httpRequestPreparer); ok {
			c.reqPreparer = v
		}
		if v, ok := option.(func(ctx context.Context, client *http.Client, req *http.Request)); ok {
			c.prepReqFunc = v
		}
		if v, ok := option.(ReaderProgressReporter); ok {
			c.rpr = v
		}
		if v, ok := option.(BoundedProgressReporter); ok {
			c.bpr = v
		}
		if v, ok := option.(tidyHandler); ok {
			c.tidyHandler = v
		}
	}

	if c.client == nil {
		c.client = &http.Client{
			Timeout: time.Second * 90,
		}
	}
}

func (c *Collection) prepareHTTPRequest(ctx context.Context, req *http.Request) {
	if c.reqPreparer != nil {
		c.reqPreparer.OnPrepareHTTPRequest(ctx, c.client, req)
	}

	if c.prepReqFunc != nil {
		c.prepReqFunc(ctx, c.client, req)
	}
}

func (c *Collection) finalize(ctx context.Context) {
	for index, item := range c.Items {
		item.finalize(ctx, c.tidyHandler, uint(index))
	}
}

// ImportCollection takes a Dropmark apiEndpoint and creates a Collection object
func ImportCollection(ctx context.Context, apiEndpoint string, options ...interface{}) (*Collection, error) {
	result := new(Collection)
	result.initOptions(ctx, apiEndpoint, options...)

	req, reqErr := http.NewRequest(http.MethodGet, apiEndpoint, nil)
	if reqErr != nil {
		return nil, fmt.Errorf("Unable to create HTTP request: %v", reqErr)
	}

	result.prepareHTTPRequest(ctx, req)

	resp, getErr := result.client.Do(req)
	if getErr != nil {
		return nil, fmt.Errorf("Unable to execute HTTP GET request: %v", getErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Dropmark API status is not HTTP OK (200): %v", resp.StatusCode)
	}

	var body []byte
	var readErr error
	reader := result.rpr.StartReportableReaderActivityInBytes(ctx, fmt.Sprintf("Processing Dropmark API request %q (%d bytes)", apiEndpoint, resp.ContentLength), resp.ContentLength, resp.Body)
	body, readErr = ioutil.ReadAll(reader)

	if readErr != nil {
		return nil, fmt.Errorf("Unable to read body from HTTP response: %v", readErr)
	}

	json.Unmarshal(body, result)
	result.rpr.CompleteReportableActivityProgress(ctx, fmt.Sprintf("Completed Dropmark API request %q with %d items", apiEndpoint, len(result.Items)))

	result.finalize(ctx)

	return result, nil
}

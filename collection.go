package dropmark

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type httpRequestPreparer interface {
	OnPrepareHTTPRequest(ctx context.Context, client *http.Client, req *http.Request)
}

type errorTracker interface {
	OnError(ctx context.Context, code string, err error)
}

type warningTracker interface {
	OnWarning(ctx context.Context, code string, message string)
}

type tidyHandler interface {
	OnTidy(ctx context.Context, tidy string)
}

type isAsynchRequested interface {
	IsAsynchRequested(ctx context.Context) bool
}

// Collection is the object returned from the Dropmark API calls after JSON unmarshalling is completed
type Collection struct {
	Name  string  `json:"name,omitempty"`
	Items []*Item `json:"items,omitempty"`

	apiEndpoint    string
	client         *http.Client
	reqPreparer    httpRequestPreparer
	prepReqFunc    func(ctx context.Context, client *http.Client, req *http.Request)
	rpr            ReaderProgressReporter
	bpr            BoundedProgressReporter
	tidyHandler    tidyHandler
	errorTracker   errorTracker
	warningTracker warningTracker
	asynch         bool
}

// ForEach satisfies the Lectio bounded content collection interface
func (c *Collection) ForEach(ctx context.Context, before func(ctx context.Context, total uint), itemHandler func(ctx context.Context, index uint, item interface{}, total uint) bool, after func(ctx context.Context, handled, total uint), options ...interface{}) {
	count := uint(len(c.Items))
	var handled uint

	if before != nil {
		before(ctx, count)
	}

	for index, content := range c.Items {
		ok := itemHandler(ctx, uint(index), content, count)
		if !ok {
			break
		}
		handled++
	}

	if after != nil {
		after(ctx, handled, count)
	}
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

func (c *Collection) initOptions(ctx context.Context, apiEndpoint string, options ...interface{}) {
	c.apiEndpoint = apiEndpoint
	c.rpr = defaultProgressReporter
	c.bpr = defaultProgressReporter

	for _, option := range options {
		if v, ok := option.(errorTracker); ok {
			c.errorTracker = v
		}
		if v, ok := option.(warningTracker); ok {
			c.warningTracker = v
		}
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
		if v, ok := option.(isAsynchRequested); ok {
			c.asynch = v.IsAsynchRequested(ctx)
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
	finalizeItem := func(ctx context.Context, index uint, item *Item) {
		item.finalize(ctx, c.tidyHandler, index)
	}

	itemsCount := len(c.Items)
	c.bpr.StartReportableActivity(ctx, fmt.Sprintf("Importing %d Dropmark Links from %q", itemsCount, c.apiEndpoint), itemsCount)
	if c.asynch {
		var wg sync.WaitGroup
		queue := make(chan int)
		for index, item := range c.Items {
			wg.Add(1)
			go func(index int, item *Item) {
				defer wg.Done()
				finalizeItem(ctx, uint(index), item)
				queue <- index
			}(index, item)
		}
		go func() {
			defer close(queue)
			wg.Wait()
		}()
		for range queue {
			c.bpr.IncrementReportableActivityProgress(ctx)
		}
	} else {
		for index, item := range c.Items {
			finalizeItem(ctx, uint(index), item)
			c.bpr.IncrementReportableActivityProgress(ctx)
		}
	}
	c.bpr.CompleteReportableActivityProgress(ctx, fmt.Sprintf("Imported %d Dropmark Links from %q", itemsCount, c.apiEndpoint))
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

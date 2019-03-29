package dropmark

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/lectio/content"
	"github.com/lectio/harvester"
	"github.com/opentracing/opentracing-go"
	"gopkg.in/jdkato/prose.v2"
)

type Title string

// Original is the title's original text
func (t Title) Original() string {
	return string(t)
}

var sourceNameAsSuffixRegEx = regexp.MustCompile(` \| .*$`) // Removes " | Healthcare IT News" from a title like "xyz title | Healthcare IT News"

// Clean is the title's "cleaned up text" (which removes "| ..."" suffixes)
func (t Title) Clean() string {
	return sourceNameAsSuffixRegEx.ReplaceAllString(string(t), "")
}

type Summary struct {
	item     *Item
	original string
}

// Original is the summary's original text (from Dropmark description)
func (s Summary) Original() string {
	return s.original
}

// FirstSentenceOfBody uses NLP to get the first sentence of the body
func (s Summary) FirstSentenceOfBody() (string, error) {
	content, proseErr := prose.NewDocument(s.item.Body())
	if proseErr != nil {
		return "", proseErr
	}

	sentences := content.Sentences()
	if len(sentences) > 0 {
		return sentences[0].Text, nil
	} else {
		return "", errors.New("Unable to find any sentences in the body")
	}
}

// OpenGraphDescription uses the HarvestedResource's open graph content if available
func (s Summary) OpenGraphDescription() (string, bool) {
	return s.item.OpenGraphContent("description", nil)
}

// HTTPUserAgent may be passed into GetDropmarkCollection as the default HTTP User-Agent header parameter
const HTTPUserAgent = "github.com/lectio/dropmark"

// HTTPTimeout may be passed into the GetDropmarkCollection as the default timeout parameter
const HTTPTimeout = time.Second * 90

// Collection is the object returned from the Dropmark API calls after JSON unmarshalling is completed
type Collection struct {
	Name        string  `json:"name,omitempty"`
	Items       []*Item `json:"items,omitempty"`
	APIEndpoint string  `json:"-"`
}

// Source returns the Dropmark API endpoint which created the collection
func (c Collection) Source() string {
	return c.APIEndpoint
}

// Content returns Dropmark items as a content collection
func (c Collection) Content() []content.Content {
	result := make([]content.Content, len(c.Items))
	for i := 0; i < len(c.Items); i++ {
		result[i] = c.Items[i]
	}
	return result
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
	Link          string      `json:"link,omitempty"`
	Name          string      `json:"name,omitempty"`
	Description   string      `json:"description,omitempty"`
	Content       string      `json:"content,omitempty"`
	Tags          []*Tag      `json:"tags,omitempty"`
	CreatedAt     string      `json:"created_at,omitempty"`
	UpdatedAt     string      `json:"updated_at,omitempty"`
	ThumbnailURL  string      `json:"thumbnail,omitempty"`
	Thumbnails    *Thumbnails `json:"thumbnails,omitempty"`
	UserID        string      `json:"user_id,omitempty"`
	UserNameShort string      `json:"username,omitempty"`
	UserNameLong  string      `json:"user_name,omitempty"`
	UserEmail     string      `json:"user_email,omitempty"`
	UserAvatarURL *Thumbnails `json:"user_avatar,omitempty"`

	title            Title
	summary          Summary
	targetURL        *url.URL
	categories       []string
	createdOn        time.Time
	featuredImageURL *url.URL
	contentKeys      content.Keys
	resource         *harvester.HarvestedResource
}

func (i *Item) init(ch *harvester.ContentHarvester, parentSpan opentracing.Span) {
	resources := ch.HarvestResources(i.Link, parentSpan).Resources
	i.resource = resources[0]
	i.title = Title(i.Name)
	i.summary = Summary{item: i, original: i.Description}
	i.categories = make([]string, len(i.Tags))
	for t := 0; t < len(i.Tags); t++ {
		i.categories[t] = i.Tags[t].Name
	}

	i.targetURL, _ = url.Parse(i.Link)
	i.createdOn, _ = time.Parse("2006-01-02 15:04:05 MST", i.CreatedAt)
	i.featuredImageURL, _ = url.Parse(i.Thumbnails.Large)
	i.contentKeys = content.CreateKeys(i, content.KeyDoesNotExist)

	_, contentURLErr := url.Parse(i.Content)
	if contentURLErr == nil {
		// Sometimes in Dropmark, the content is just a URL (not sure why).
		// If the content is just a single URL, replace it with the Description
		i.Content = i.Description
	}
}

func (i Item) Title() content.Title {
	return i.title
}

func (i Item) Body() string {
	return i.Content
}

func (i Item) Summary() content.Summary {
	return i.summary
}

func (i Item) Categories() []string {
	return i.categories
}

func (i Item) CreatedOn() time.Time {
	return i.createdOn
}

func (i Item) FeaturedImage() *url.URL {
	return i.featuredImageURL
}

func (i Item) Keys() content.Keys {
	return i.contentKeys
}

// OpenGraphContent uses the HarvestedResource's open graph content if available
func (i Item) OpenGraphContent(ogKey string, defaultValue *string) (string, bool) {
	rc := i.resource.ResourceContent()
	if rc == nil {
		if defaultValue == nil {
			return "", false
		}
		return *defaultValue, true
	}
	return rc.GetOpenGraphMetaTag(ogKey)
}

// TwitterContent uses the content's TwitterCard meta data
func (i Item) TwitterCardContent(twitterKey string, defaultValue *string) (string, bool) {
	rc := i.resource.ResourceContent()
	if rc == nil {
		if defaultValue == nil {
			return "", false
		}
		return *defaultValue, true
	}
	return rc.GetTwitterMetaTag(twitterKey)
}

func (i Item) Target() *url.URL {
	return i.targetURL
}

// GetDropmarkCollection takes a Dropmark apiEndpoint and creates a Collection object
func GetDropmarkCollection(ch *harvester.ContentHarvester, parentSpan opentracing.Span, apiEndpoint string, userAgent string, timeout time.Duration) (*Collection, error) {
	result := new(Collection)
	result.APIEndpoint = apiEndpoint

	httpClient := http.Client{
		Timeout: timeout,
	}
	req, reqErr := http.NewRequest(http.MethodGet, apiEndpoint, nil)
	if reqErr != nil {
		return nil, fmt.Errorf("Unable to create request %q: %v", apiEndpoint, reqErr)
	}
	req.Header.Set("User-Agent", userAgent)
	res, getErr := httpClient.Do(req)
	if getErr != nil {
		return nil, fmt.Errorf("Unable to execute GET request %q: %v", apiEndpoint, getErr)
	}
	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		return nil, fmt.Errorf("Unable to read body from request %q: %v", apiEndpoint, readErr)
	}

	json.Unmarshal(body, result)

	if result.Items != nil {
		for i := 0; i < len(result.Items); i++ {
			result.Items[i].init(ch, parentSpan)
		}
	}

	return result, nil
}

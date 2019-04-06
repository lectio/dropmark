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
	"github.com/lectio/flexmap"
	"github.com/lectio/frontmatter"
	"github.com/lectio/link"
	"gopkg.in/cheggaaa/pb.v1"
	"gopkg.in/jdkato/prose.v2"
)

// DropmarkEditorURLDirectiveName is the content.Content.Directive() key for the editor URL
const DropmarkEditorURLDirectiveName = "editorURL"

// Title defines a Dropmark item's title in various formats
type Title struct {
	item     *Item
	original string
}

// Original is the title's original text
func (t Title) Original() string {
	return t.original
}

var sourceNameAsSuffixRegEx = regexp.MustCompile(` \| .*$`) // Removes " | Healthcare IT News" from a title like "xyz title | Healthcare IT News"

// Clean is the title's "cleaned up text" (which removes "| ..."" suffixes)
func (t Title) Clean() string {
	return sourceNameAsSuffixRegEx.ReplaceAllString(t.original, "")
}

// OpenGraphTitle uses the HarvestedResource's open graph title if available
func (t Title) OpenGraphTitle(clean bool) (string, bool) {
	title, ok := t.item.OpenGraphContent("title", nil)
	if ok {
		if clean {
			return sourceNameAsSuffixRegEx.ReplaceAllString(title, ""), true
		}
		return title, true
	}
	return "", false
}

// Summary defines a Dropmark item's description/summary in various formats
type Summary struct {
	item     *Item
	original string
}

// Original is the summary's original text (from Dropmark description)
func (s Summary) Original() string {
	return s.original
}

// OpenGraphDescription uses the HarvestedResource's open graph description if available
func (s Summary) OpenGraphDescription() (string, bool) {
	return s.item.OpenGraphContent("description", nil)
}

// Body defines a Dropmark item's primary content in various formats
type Body struct {
	item               *Item
	original           string
	replacedByDescr    bool
	haveFrontMatter    bool
	frontMatter        flexmap.TextKeyMap
	withoutFrontMatter string
}

func makeBody(c *Collection, index int, item *Item) Body {
	result := Body{}
	result.original = item.Content
	_, contentURLErr := url.Parse(item.Content)
	if contentURLErr == nil {
		// Sometimes in Dropmark, the content is just a URL (not sure why).
		// If the entire content is just a single URL, replace it with the Description
		result.original = item.Description
		result.replacedByDescr = true
	}

	var body []byte
	var haveFrontMatter bool
	frontMatter, fmErr := flexmap.MakeCustomTextKeyMap(func(v interface{}) error {
		var fmErr error
		body, haveFrontMatter, fmErr = frontmatter.ParseYAMLFrontMatter([]byte(result.original), v)
		return fmErr
	})
	if fmErr != nil {
		result.haveFrontMatter = false
		item.addError(c, fmt.Errorf("harvested Dropmark resource item %d body front matter error: %v", index, fmErr))
	} else if haveFrontMatter {
		result.haveFrontMatter = true
		result.frontMatter = frontMatter
		result.withoutFrontMatter = fmt.Sprintf("%s", body)
	}
	return result
}

// Original returns the body content as supplied by Dropmark
func (b Body) Original() string {
	return b.original
}

// FirstSentence uses NLP to get the first sentence of the body
func (b Body) FirstSentence() (string, error) {
	content, proseErr := prose.NewDocument(b.WithoutFrontMatter())
	if proseErr != nil {
		return "", proseErr
	}

	sentences := content.Sentences()
	if len(sentences) > 0 {
		return sentences[0].Text, nil
	}
	return "", errors.New("Unable to find any sentences in the body")
}

// WithoutFrontMatter returns either the original body or any content past the front matter (if any)
func (b Body) WithoutFrontMatter() string {
	if b.haveFrontMatter {
		return b.withoutFrontMatter
	}
	return b.original
}

// HaveFrontMatter returns true if any front matter was found as part of the body
func (b Body) HaveFrontMatter() bool {
	return b.haveFrontMatter
}

// FrontMatter returns a map of key value pairs at the start of the body, if any
func (b Body) FrontMatter() flexmap.TextKeyMap {
	return b.frontMatter
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
	errors      []error
}

func (c *Collection) init(cleanCurationTargetRule link.CleanResourceParamsRule, ignoreCurationTargetRule link.IgnoreResourceRule,
	followHTMLRedirect link.FollowRedirectsInCurationTargetHTMLPayload, verbose bool) {
	var bar *pb.ProgressBar
	if verbose {
		bar = pb.StartNew(len(c.Items))
		bar.ShowCounters = true
	}
	ch := make(chan int)
	if c.Items != nil {
		for i := 0; i < len(c.Items); i++ {
			go c.Items[i].init(c, i, ch, cleanCurationTargetRule, ignoreCurationTargetRule, followHTMLRedirect)
		}
	}

	for i := 0; i < len(c.Items); i++ {
		_ = <-ch
		if verbose {
			bar.Increment()
		}
	}

	if verbose {
		bar.FinishPrint(fmt.Sprintf("Completed parsing %d Dropmark items from %q", len(c.Items), c.apiEndpoint))
	}
}

// Source returns the Dropmark API endpoint which created the collection
func (c Collection) Source() string {
	return c.apiEndpoint
}

// Content returns Dropmark items as a content collection
func (c Collection) Content() []content.Content {
	result := make([]content.Content, len(c.Items))
	for i := 0; i < len(c.Items); i++ {
		result[i] = c.Items[i]
	}
	return result
}

func (c *Collection) addError(err error) {
	c.errors = append(c.errors, err)
}

// Errors returns any errors reported at the collection level
func (c Collection) Errors() []error {
	return c.errors
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

	index            int
	title            Title
	summary          Summary
	body             Body
	targetURL        *url.URL
	categories       []string
	createdOn        time.Time
	featuredImageURL *url.URL
	resource         *link.Resource
	errors           []error
	directives       flexmap.Map
}

func (i *Item) init(c *Collection, index int, ch chan<- int, cleanCurationTargetRule link.CleanResourceParamsRule, ignoreCurationTargetRule link.IgnoreResourceRule,
	followHTMLRedirect link.FollowRedirectsInCurationTargetHTMLPayload) {
	i.directives, _ = flexmap.MakeTextKeyMapWithDefaults(func() (startIndex int, endIndex int, iteratorFn func(index int) (flexmap.Item, bool)) {
		return 0, 0, func(index int) (flexmap.Item, bool) {
			return flexmap.MakeItem(DropmarkEditorURLDirectiveName, i.DropmarkEditURL), false
		}
	})

	i.body = makeBody(c, index, i)
	if i.body.HaveFrontMatter() {
		i.body.FrontMatter().ForEachTextKey(func(key string, value interface{}) bool {
			switch key {
			case "description":
				i.Description = value.(string)
			default:
				i.addError(c, fmt.Errorf("harvested Dropmark resource item %d body front matter key %s not handled", index, key))
			}
			return true
		})
	}
	i.index = index
	i.resource = link.HarvestResource(i.Link, cleanCurationTargetRule, ignoreCurationTargetRule, followHTMLRedirect)
	if i.resource == nil {
		i.addError(c, fmt.Errorf("unable to harvest Dropmark item %d link %q, resource came back nil", index, i.Link))
	}
	i.title = Title{item: i, original: i.Name}
	i.summary = Summary{item: i, original: i.Description}
	i.categories = make([]string, len(i.Tags))
	for t := 0; t < len(i.Tags); t++ {
		i.categories[t] = i.Tags[t].Name
	}

	if i.resource != nil {
		isURLValid, isDestValid := i.resource.IsValid()
		if isURLValid && isDestValid {
			i.targetURL, _, _ = i.resource.GetURLs()
		} else {
			isIgnored, ignoreReason := i.resource.IsIgnored()
			i.addError(c, fmt.Errorf("harvested Dropmark resource item %d link %q was not valid, isURLValid: %v, isDestValid: %v, isIgnored: %v, reason: %v", index, i.Link, isURLValid, isDestValid, isIgnored, ignoreReason))
			i.targetURL, _ = url.Parse(i.Link)
		}
	} else {
		i.targetURL, _ = url.Parse(i.Link)
	}
	i.createdOn, _ = time.Parse("2006-01-02 15:04:05 MST", i.CreatedAt)
	i.featuredImageURL, _ = url.Parse(i.Thumbnails.Large)

	ch <- index
}

func (i *Item) addError(c *Collection, err error) {
	i.errors = append(i.errors, err)
	c.addError(err)
}

// Errors returns any errors reported at the Item level
func (i Item) Errors() []error {
	return i.errors
}

// Keys returns a Dropmark item's different keys
func (i Item) Keys() content.Keys {
	return i.resource
}

// Title returns a Dropmark item's title in various formats
func (i Item) Title() content.Title {
	return i.title
}

// Body returns a Dropmark item's main content
func (i Item) Body() content.Body {
	return i.body
}

// Summary returns a Dropmark item's title in various formats
func (i Item) Summary() content.Summary {
	return i.summary
}

// Categories returns a Dropmark item's tags
func (i Item) Categories() []string {
	return i.categories
}

// CreatedOn returns a Dropmark item's creation date
func (i Item) CreatedOn() time.Time {
	return i.createdOn
}

// FeaturedImage returns a Dropmark item's primary image URL
func (i Item) FeaturedImage() *url.URL {
	return i.featuredImageURL
}

// Directives returns a Dropmark item's annotations, pragmas, and directives
func (i Item) Directives() flexmap.Map {
	return i.directives
}

// OpenGraphContent uses the HarvestedResource's open graph content if available
func (i Item) OpenGraphContent(ogKey string, defaultValue *string) (string, bool) {
	if i.resource == nil {
		if defaultValue == nil {
			return "", false
		}
		return *defaultValue, true
	}
	ir := i.resource.InspectionResults()
	if ir == nil {
		if defaultValue == nil {
			return "", false
		}
		return *defaultValue, true
	}
	return ir.GetOpenGraphMetaTag(ogKey)
}

// TwitterCardContent uses the content's TwitterCard meta data
func (i Item) TwitterCardContent(twitterKey string, defaultValue *string) (string, bool) {
	if i.resource == nil {
		if defaultValue == nil {
			return "", false
		}
		return *defaultValue, true
	}
	ir := i.resource.InspectionResults()
	if ir == nil {
		if defaultValue == nil {
			return "", false
		}
		return *defaultValue, true
	}
	return ir.GetTwitterMetaTag(twitterKey)
}

// TargetResource is the URL that Dropmark item points to
func (i Item) TargetResource() *link.Resource {
	return i.resource
}

// GetDropmarkCollection takes a Dropmark apiEndpoint and creates a Collection object
func GetDropmarkCollection(apiEndpoint string, cleanCurationTargetRule link.CleanResourceParamsRule, ignoreCurationTargetRule link.IgnoreResourceRule,
	followHTMLRedirect link.FollowRedirectsInCurationTargetHTMLPayload, verbose bool, userAgent string, timeout time.Duration) (*Collection, error) {
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
	if verbose {
		bar := pb.New(int(resp.ContentLength)).SetUnits(pb.U_BYTES)
		bar.Start()
		reader := bar.NewProxyReader(resp.Body)
		body, readErr = ioutil.ReadAll(reader)
		bar.FinishPrint(fmt.Sprintf("Completed Dropmark API request %q", apiEndpoint))
	} else {
		body, readErr = ioutil.ReadAll(resp.Body)
	}

	if readErr != nil {
		return nil, fmt.Errorf("Unable to read body from request %q: %v", apiEndpoint, readErr)
	}

	json.Unmarshal(body, result)
	result.init(cleanCurationTargetRule, ignoreCurationTargetRule, followHTMLRedirect, verbose)
	return result, nil
}

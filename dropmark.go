package dropmark

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// HTTPUserAgent may be passed into GetDropmarkCollection as the default HTTP User-Agent header parameter
const HTTPUserAgent = "github.com/lectio/dropmark"

// HTTPTimeout may be passed into the GetDropmarkCollection as the default timeout parameter
const HTTPTimeout = time.Second * 90

// Collection is the object return from the Drop API calls after JSON unmarshalling is completed
type Collection struct {
	Name        string  `json:"name,omitempty"`
	Items       []*Item `json:"items,omitempty"`
	APIEndpoint string  `json:"-"`
}

// Item represents a single Dropmark collection item after JSON unmarshalling is completed
type Item struct {
	Link          string      `json:"link,omitempty"`
	Title         string      `json:"name,omitempty"`
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

// GetDropmarkCollection takes a Dropmark apiEndpoint and creates a Collection object
func GetDropmarkCollection(apiEndpoint string, userAgent string, timeout time.Duration) (*Collection, error) {
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
	return result, nil
}

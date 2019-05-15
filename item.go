package dropmark

import (
	"fmt"
	"net/url"
	"strings"
)

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
	ID              string      `json:"id"`
	IsURL           bool        `json:"is_url"`
	Type            string      `json:"type"`
	MIME            string      `json:"mime"`
	Link            string      `json:"link,omitempty"`
	Name            string      `json:"name,omitempty"`
	Description     string      `json:"description,omitempty"`
	Content         string      `json:"content,omitempty"`
	Tags            []*Tag      `json:"tags,omitempty"`
	CreatedAt       string      `json:"created_at,omitempty"`
	UpdatedAt       string      `json:"updated_at,omitempty"`
	DeletedAt       string      `json:"deleted_at,omitempty"`
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

// OriginalURL satisfies the contract for a Lectio link.Link object
func (i *Item) OriginalURL() string {
	return i.Link
}

// FinalURL satisfies the contract for a Lectio link.Link object
func (i *Item) FinalURL() (*url.URL, error) {
	return url.Parse(i.Link)
}

// Edits returns a list of messages indicating what edits, if any were performed on the Dropmark URLs
func (i *Item) Edits() []string {
	return i.edits
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

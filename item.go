package dropmark

import (
	"context"
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
	ContentArchetype string      `json:"archetype"`             // injected by Lectio in finalize()
	Index            uint        `json:"index_in_collection"`   // injected by Lectio in finalize()
	ID               string      `json:"id"`                    // from Dropmark API
	IsURL            bool        `json:"is_url"`                // from Dropmark API
	Type             string      `json:"type"`                  // from Dropmark API
	MIME             string      `json:"mime"`                  // from Dropmark API
	Link             string      `json:"link,omitempty"`        // from Dropmark API
	Name             string      `json:"name,omitempty"`        // from Dropmark API
	Description      string      `json:"description,omitempty"` // from Dropmark API
	Content          string      `json:"content,omitempty"`     // from Dropmark API
	Tags             []*Tag      `json:"tags,omitempty"`        // from Dropmark API
	CreatedAt        string      `json:"created_at,omitempty"`  // from Dropmark API
	UpdatedAt        string      `json:"updated_at,omitempty"`  // from Dropmark API
	DeletedAt        string      `json:"deleted_at,omitempty"`  // from Dropmark API
	ThumbnailURL     string      `json:"thumbnail,omitempty"`   // from Dropmark API
	Thumbnails       *Thumbnails `json:"thumbnails,omitempty"`  // from Dropmark API
	UserID           string      `json:"user_id,omitempty"`     // from Dropmark API
	UserNameShort    string      `json:"username,omitempty"`    // from Dropmark API
	UserNameLong     string      `json:"user_name,omitempty"`   // from Dropmark API
	UserEmail        string      `json:"user_email,omitempty"`  // from Dropmark API
	UserAvatarURL    *Thumbnails `json:"user_avatar,omitempty"` // from Dropmark API
	DropmarkEditURL  string      `json:"url"`                   // from Dropmark API

	finalized bool
}

// OriginalURL satisfies the contract for a Lectio link.Link object
func (i *Item) OriginalURL() string {
	return i.Link
}

// FinalURL satisfies the contract for a Lectio link.Link object
func (i *Item) FinalURL() (*url.URL, error) {
	var warning string
	warn := func(code, message string) {
		warning = fmt.Sprintf("[%s] %s", code, message)
	}

	if i.Traversable(warn) {
		return url.Parse(i.OriginalURL())
	}
	return nil, fmt.Errorf(warning)
}

// Traversable satisfies the contract for a Lectio link.Link object
func (i *Item) Traversable(warn func(code, message string)) bool {
	if len(i.DeletedAt) > 0 {
		warn("DMIWARN-001-ITEMDELETED", "Item marked as deleted, not traversable")
		return false
	}

	if i.Type != "link" {
		warn("DMIWARN-002-ITEMNOTLINK", fmt.Sprintf("Item 'type' is %q not 'link', not traversable", i.Type))
		return false
	}

	if len(strings.TrimSpace(i.Link)) == 0 {
		warn("DMIWARN-003-LINKEMPTY", "Empty link, not traversable")
		return false
	}

	return true
}

func (i *Item) finalize(ctx context.Context, tidyInstance tidyHandler, index uint) {
	if i.finalized {
		return
	}

	i.ContentArchetype = "bookmark"
	i.Index = index

	onTidy := func(tidy string) {
		if tidyInstance != nil {
			tidyInstance.OnTidy(ctx, tidy)
		}
	}

	_, contentURLErr := url.Parse(i.Content)
	if contentURLErr == nil {
		// Sometimes in Dropmark, the content is just a URL (not sure why).
		// If the entire content is just a single URL, replace it with the Description
		onTidy(fmt.Sprintf("Item[%d].Content was a URL %q, replaced with Description", index, i.Content))
		i.Content = i.Description
	}

	if strings.Compare(i.Content, i.Description) == 0 {
		onTidy(fmt.Sprintf("Item[%d].Content was the same as the Description, set Description to blank", index))
		i.Description = ""
	}

	i.finalized = true
}

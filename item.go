package dropmark

import (
	"context"
	"fmt"
	"github.com/lectio/link"
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
	RawContent      string      `json:"content,omitempty"`
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

	index              uint
	edits              []string
	linkTraversed      bool
	linkTraversable    bool
	traversedLink      link.Link
	linkTraversalError error

	finalized bool
}

// OriginalURL satisfies the contract for a Lectio link.Link object
func (i *Item) OriginalURL(ctx context.Context, options ...interface{}) string {
	return i.Link
}

// FinalURL satisfies the contract for a Lectio link.Link object
func (i *Item) FinalURL(ctx context.Context, options ...interface{}) (*url.URL, error) {
	if i.linkTraversed && i.linkTraversable {
		return i.traversedLink.FinalURL()
	}
	return url.Parse(i.OriginalURL(ctx, options...))
}

// TraversedLink satisfies the contract for a Lectio Content interface
func (i *Item) TraversedLink(ctx context.Context, options ...interface{}) (bool, bool, link.Link, error) {
	return i.linkTraversable, i.linkTraversed, i.traversedLink, i.linkTraversalError
}

func (i *Item) isTraversable(ctx context.Context, warn func(code, message string)) bool {
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

// TraversedLink satisfies the contract for a Lectio Content interface
func (i *Item) traverseLink(ctx context.Context, traversable func(item *Item) bool, traverse func(item *Item) (link.Link, error)) {
	if i.linkTraversed {
		return
	}

	i.linkTraversable = traversable(i)
	if i.linkTraversable {
		i.traversedLink, i.linkTraversalError = traverse(i)
	}
	i.linkTraversed = true
}

// Title satisfies the contract for a Lectio Content interface
func (i *Item) Title(ctx context.Context, options ...interface{}) string {
	return i.Name
}

// Summary satisfies the contract for a Lectio Content interface
func (i *Item) Summary(ctx context.Context, options ...interface{}) string {
	return i.Description
}

// Content satisfies the contract for a Lectio Content interface
func (i *Item) Content(ctx context.Context, options ...interface{}) string {
	return i.RawContent
}

// FeaturedImageURL satisfies the contract for a Lectio Content interface
func (i *Item) FeaturedImageURL(ctx context.Context, options ...interface{}) string {
	return i.ThumbnailURL
}

func (i *Item) finalize(ctx context.Context, tidyInstance tidyHandler, index uint) {
	if i.finalized {
		return
	}

	i.index = index

	onTidy := func(tidy string) {
		i.edits = append(i.edits, tidy)
		if tidyInstance != nil {
			tidyInstance.OnTidy(ctx, tidy)
		}
	}

	_, contentURLErr := url.Parse(i.RawContent)
	if contentURLErr == nil {
		// Sometimes in Dropmark, the content is just a URL (not sure why).
		// If the entire content is just a single URL, replace it with the Description
		onTidy(fmt.Sprintf("Item[%d].Content was a URL %q, replaced with Description", index, i.RawContent))
		i.RawContent = i.Description
	}

	if strings.Compare(i.RawContent, i.Description) == 0 {
		onTidy(fmt.Sprintf("Item[%d].Content was the same as the Description, set Description to blank", index))
		i.Description = ""
	}

	i.finalized = true
}

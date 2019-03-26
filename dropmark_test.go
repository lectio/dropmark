package dropmark

import (
	"testing"
)

func TestDropmarkCollection(t *testing.T) {
	collection, getErr := GetDropmarkCollection("https://shah.dropmark.com/616548.json", HTTPUserAgent, HTTPTimeout)
	if getErr != nil {
		t.Errorf("Unable to retrieve Dropmark collection from %q: %v.", collection.APIEndpoint, getErr)
	} else {
		t.Logf("Retrieved %d items from %q", len(collection.Content()), collection.APIEndpoint)
	}
}

package dropmark

import (
	"context"
	"regexp"
)

var (
	apiPattern = regexp.MustCompile(`^https\://(.*).dropmark.com/([0-9]+).json$`)
)

// IsValidAPIEndpoint tests to see if the apiEndpoint is a Dropmark API endpoint
func IsValidAPIEndpoint(apiEndpoint string) bool {
	return apiPattern.MatchString(apiEndpoint)
}

// Import satisfies the standard Lectio "import from API" function
func Import(ctx context.Context, apiEndpoint string, options ...interface{}) (interface{}, error) {
	return ImportCollection(ctx, apiEndpoint, options...)
}

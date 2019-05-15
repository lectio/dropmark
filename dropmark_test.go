package dropmark

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type DropmarkSuite struct {
	suite.Suite
	httpClient *http.Client
}

func (suite *DropmarkSuite) SetupSuite() {
	suite.httpClient = &http.Client{Timeout: time.Second * 66}
}

func (suite *DropmarkSuite) TearDownSuite() {
}

func (suite *DropmarkSuite) HTTPClient() *http.Client {
	return suite.httpClient
}

func (suite *DropmarkSuite) OnPrepareHTTPRequest(ctx context.Context, client *http.Client, req *http.Request) {
	req.Header.Set("User-Agent", "github.com/lectio/dropmark.DropmarkSuite")
}

func (suite *DropmarkSuite) StartReportableReaderActivityInBytes(summary string, exepectedBytes int64, inputReader io.Reader) io.Reader {
	return inputReader
}

func (suite *DropmarkSuite) CompleteReportableActivityProgress(summary string) {

}

func (suite *DropmarkSuite) TestDropmarkCollection() {
	ctx := context.Background()
	collection, getErr := GetCollection(ctx, "https://shah.dropmark.com/652682.json", suite)
	suite.Nil(getErr, "Unable to retrieve Dropmark collection from %q: %v.", collection.apiEndpoint, getErr)
	suite.Equal(len(collection.Items), 4)
}

func (suite *DropmarkSuite) TestContent() {
	ctx := context.Background()
	collection, getErr := GetCollection(ctx, "https://shah.dropmark.com/652682.json")
	suite.Nil(getErr, "Unable to retrieve Dropmark collection from %q: %v.", collection.apiEndpoint, getErr)
	count, getItemFn, contentErr := collection.Content()
	suite.Nil(contentErr, "Unable to get Dropmark content iterator from %q: %v.", collection.apiEndpoint, contentErr)
	suite.Equal(count, 4)

	// get a specific item -- the item function will return a single item
	genericItem1, genericItem1Err := getItemFn(1, 1)
	suite.Nil(genericItem1Err, "Unable to get Dropmark content item from %q: %v.", collection.apiEndpoint, genericItem1Err)
	item1 := genericItem1.(*Item)
	suite.Equal("https://www2.deloitte.com/insights/us/en/industry/financial-services/demystifying-cybersecurity-insurance.html", item1.OriginalURL())

	// get a range of items -- the item function will return a slice
	genericItems, genericItemsErr := getItemFn(1, 3)
	suite.Nil(genericItem1Err, "Unable to get Dropmark content items from %q: %v.", collection.apiEndpoint, genericItemsErr)
	items := genericItems.([]*Item)
	suite.Equal(3, len(items))
}

func (suite *DropmarkSuite) TestInvalid() {
	ctx := context.Background()
	_, getErr := GetCollection(ctx, "https://sha.dropmark.com/652682.json")
	suite.NotNil(getErr, "Should be an error", getErr)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(DropmarkSuite))
}

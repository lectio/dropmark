package dropmark

import (
	"context"
	"github.com/lectio/link"
	"github.com/lectio/progress"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type DropmarkSuite struct {
	suite.Suite
	httpClient *http.Client
	factory    link.Factory
}

func (suite *DropmarkSuite) SetupSuite() {
	suite.httpClient = &http.Client{Timeout: time.Second * 66}
	suite.factory = link.NewFactory(suite)
}

func (suite *DropmarkSuite) TearDownSuite() {
}

func (suite *DropmarkSuite) HTTPClient(ctx context.Context) *http.Client {
	return suite.httpClient
}

func (suite *DropmarkSuite) OnPrepareHTTPRequest(ctx context.Context, client *http.Client, req *http.Request) {
	req.Header.Set("User-Agent", "github.com/lectio/dropmark.DropmarkSuite")
}

func (suite *DropmarkSuite) StartReportableReaderActivityInBytes(ctx context.Context, summary string, exepectedBytes int64, inputReader io.Reader) io.Reader {
	return inputReader
}

func (suite *DropmarkSuite) CompleteReportableActivityProgress(ctx context.Context, summary string) {

}

func (suite *DropmarkSuite) TestDropmarkCollection() {
	spr := progress.NewSummaryReporter("TestDropmarkCollection()")
	ctx := context.Background()
	collection, getErr := ImportCollection(ctx, "https://shah.dropmark.com/652682.json", suite, spr, suite.factory)
	suite.Nil(getErr, "Unable to retrieve Dropmark collection from %q: %v.", collection.APIEndpoint, getErr)
	suite.Equal(len(collection.Items), 4)
}

func (suite *DropmarkSuite) TestInvalid() {
	ctx := context.Background()
	_, getErr := ImportCollection(ctx, "https://sha.dropmark.com/652682.json", suite, suite.factory)
	suite.NotNil(getErr, "Should be an error", getErr)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(DropmarkSuite))
}

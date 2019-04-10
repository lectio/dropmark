package dropmark

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DropmarkSuite struct {
	suite.Suite
}

func (suite *DropmarkSuite) SetupSuite() {
}

func (suite *DropmarkSuite) TearDownSuite() {
}

func (suite *DropmarkSuite) TestDropmarkCollection() {
	collection, getErr := GetCollection("https://shah.dropmark.com/652682.json", nil, HTTPUserAgent, HTTPTimeout)
	suite.Nil(getErr, "Unable to retrieve Dropmark collection from %q: %v.", collection.apiEndpoint, getErr)
	suite.Equal(len(collection.Items), 4)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(DropmarkSuite))
}

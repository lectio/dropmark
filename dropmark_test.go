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

func (suite *DropmarkSuite) TestContent() {
	collection, getErr := GetCollection("https://shah.dropmark.com/652682.json", nil, HTTPUserAgent, HTTPTimeout)
	suite.Nil(getErr, "Unable to retrieve Dropmark collection from %q: %v.", collection.apiEndpoint, getErr)
	count, getItemFn, contentErr := collection.Content()
	suite.Nil(contentErr, "Unable to get Dropmark content iterator from %q: %v.", collection.apiEndpoint, contentErr)
	suite.Equal(count, 4)

	// get a specific item -- the item function will return a single item
	genericItem1, genericItem1Err := getItemFn(1, 1)
	suite.Nil(genericItem1Err, "Unable to get Dropmark content item from %q: %v.", collection.apiEndpoint, genericItem1Err)
	item1 := genericItem1.(*Item)
	suite.Equal("https://www2.deloitte.com/insights/us/en/industry/financial-services/demystifying-cybersecurity-insurance.html", item1.TargetURL())

	// get a range of items -- the item function will return a slice
	genericItems, genericItemsErr := getItemFn(1, 3)
	suite.Nil(genericItem1Err, "Unable to get Dropmark content items from %q: %v.", collection.apiEndpoint, genericItemsErr)
	items := genericItems.([]*Item)
	suite.Equal(3, len(items))
}

func (suite *DropmarkSuite) TestInvalid() {
	_, getErr := GetCollection("https://sha.dropmark.com/652682.json", nil, HTTPUserAgent, HTTPTimeout)
	suite.NotNil(getErr, "Should be an error", getErr)
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(DropmarkSuite))
}

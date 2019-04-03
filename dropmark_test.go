package dropmark

import (
	"fmt"
	"net/url"
	"regexp"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ignoreURLsRegExList []*regexp.Regexp
type removeParamsFromURLsRegExList []*regexp.Regexp

var defaultIgnoreURLsRegExList ignoreURLsRegExList = []*regexp.Regexp{regexp.MustCompile(`^https://twitter.com/(.*?)/status/(.*)$`), regexp.MustCompile(`https://t.co`)}
var defaultCleanURLsRegExList removeParamsFromURLsRegExList = []*regexp.Regexp{regexp.MustCompile(`^utm_`)}

func (l ignoreURLsRegExList) IgnoreResource(url *url.URL) (bool, string) {
	URLtext := url.String()
	for _, regEx := range l {
		if regEx.MatchString(URLtext) {
			return true, fmt.Sprintf("Matched Ignore Rule `%s`", regEx.String())
		}
	}
	return false, ""
}

func (l removeParamsFromURLsRegExList) CleanResourceParams(url *url.URL) bool {
	// we try to clean all URLs, not specific ones
	return true
}

func (l removeParamsFromURLsRegExList) RemoveQueryParamFromResourceURL(paramName string) (bool, string) {
	for _, regEx := range l {
		if regEx.MatchString(paramName) {
			return true, fmt.Sprintf("Matched cleaner rule `%s`", regEx.String())
		}
	}

	return false, ""
}

type DropmarkSuite struct {
	suite.Suite
}

func (suite *DropmarkSuite) SetupSuite() {
}

func (suite *DropmarkSuite) TearDownSuite() {
}

func (suite *DropmarkSuite) TestDropmarkCollection() {
	collection, getErr := GetDropmarkCollection("https://shah.dropmark.com/652682.json", defaultCleanURLsRegExList, defaultIgnoreURLsRegExList, true, false, HTTPUserAgent, HTTPTimeout)
	suite.Nil(getErr, "Unable to retrieve Dropmark collection from %q: %v.", collection.apiEndpoint, getErr)
	suite.Nil(collection.Errors(), "There should be no errors at the collection level")
	suite.Equal(len(collection.Items), 4)

	item := collection.Items[0]
	suite.Equal(item.Title().Original(), "University Health System to invest $170 million in new medical record technology")
	suite.Equal(item.Summary().Original(), "Bexar County’s public hospital district is spending $170 million to update its health information technology — an investment that administrators say could save the system millions of dollars and improve patient care.")
	suite.Nil(item.Errors(), "There should be no errors at the item level")

	item = collection.Items[1]
	suite.Equal(item.Title().Original(), "Demystifying cybersecurity insurance | Deloitte Insights")
	suite.Equal(item.Title().Clean(), "Demystifying cybersecurity insurance")
	suite.Equal(item.Summary().Original(), "\u200bOrganizations continue to invest heavily in cybersecurity efforts to safeguard themselves against threats, but far fewer have signed on for cyber insurance to protect their firms\u00a0afteran attack. Why not? What roadblocks exist, and what steps could the industry take to help clear them?")
	suite.Nil(item.Errors(), "There should be no errors at the item level")

	title, _ := item.Title().OpenGraphTitle(true)
	suite.Equal(title, "Demystifying cyber insurance coverage")

	descr, _ := item.Summary().OpenGraphDescription()
	suite.True(len(descr) > 0, "Description should be available as og:description in <meta>")

	item = collection.Items[2]
	suite.Equal(item.Title().Original(), "Cybersecurity As A Competitive Differentiator For Medical Devices")
	suite.Equal(item.Summary().Original(), "In 2013, the Food and Drug Administration (FDA) issued its first cybersecurity safety communication, followed in 2014 by final guidance. While it took the agency much longer to focus on cybersecurity than many of us would have liked, I think it struck a reasonable balance between new regulations (almost none) and guidance (in the form of nonbinding recommendations).")
	suite.Nil(item.Errors(), "There should be no errors at the item level")

	descr, nlpErr := item.Body().FirstSentence()
	suite.Nil(nlpErr, "Unable to retrieve first sentence of item.Summary(): %v", nlpErr)
	suite.Equal(descr, "In 2013, the Food and Drug Administration (FDA) issued its first cybersecurity safety communication, followed in 2014 by final guidance.")

	item = collection.Items[3]
	suite.Equal(item.Title().Original(), "Signify Research HIMSS 2019 Show Report (PDF)")
	suite.Nil(item.Errors(), "There should be no errors at the item level")
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(DropmarkSuite))
}

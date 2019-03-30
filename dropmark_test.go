package dropmark

import (
	"os"
	"testing"

	"github.com/lectio/harvester"
	"github.com/opentracing/opentracing-go"
	"github.com/shah/observe-go"
	"github.com/stretchr/testify/suite"
)

type DropmarkSuite struct {
	suite.Suite
	observatory  observe.Observatory
	cntHarvester *harvester.ContentHarvester
	span         opentracing.Span
}

func (suite *DropmarkSuite) SetupSuite() {
	_, set := os.LookupEnv("JAEGER_SERVICE_NAME")
	if !set {
		os.Setenv("JAEGER_SERVICE_NAME", "Lectio Harvester Test Suite")
	}

	observatory := observe.MakeObservatoryFromEnv()
	suite.observatory = observatory
	suite.span = observatory.StartTrace("DropmarkSuite")
	suite.cntHarvester = harvester.MakeContentHarvester(suite.observatory, harvester.DefaultIgnoreURLsRegExList, harvester.DefaultCleanURLsRegExList, false)
}

func (suite *DropmarkSuite) TearDownSuite() {
	suite.span.Finish()
	suite.observatory.Close()
}

func (suite *DropmarkSuite) TestDropmarkCollection() {
	collection, getErr := GetDropmarkCollection(suite.cntHarvester, suite.span, false, "https://shah.dropmark.com/652682.json", HTTPUserAgent, HTTPTimeout)
	suite.Nil(getErr, "Unable to retrieve Dropmark collection from %q: %v.", collection.apiEndpoint, getErr)
	suite.Nil(collection.Errors(), "There should be no errors at the collection level")
	suite.Equal(len(collection.Items), 3)

	item := collection.Items[0]
	suite.Equal(item.Title().Original(), "Demystifying cybersecurity insurance | Deloitte Insights")
	suite.Equal(item.Title().Clean(), "Demystifying cybersecurity insurance")
	suite.Equal(item.Summary().Original(), "\u200bOrganizations continue to invest heavily in cybersecurity efforts to safeguard themselves against threats, but far fewer have signed on for cyber insurance to protect their firms\u00a0afteran attack. Why not? What roadblocks exist, and what steps could the industry take to help clear them?")
	suite.Nil(item.Errors(), "There should be no errors at the item level")

	title, _ := item.Title().OpenGraphTitle()
	suite.Equal(title, "Demystifying cyber insurance coverage")

	descr, _ := item.Summary().OpenGraphDescription()
	suite.True(len(descr) > 0, "Description should be available as og:description in <meta>")

	item = collection.Items[1]
	suite.Equal(item.Title().Original(), "Cybersecurity As A Competitive Differentiator For Medical Devices")
	suite.Equal(item.Summary().Original(), "In 2013, the Food and Drug Administration (FDA) issued its first cybersecurity safety communication, followed in 2014 by final guidance. While it took the agency much longer to focus on cybersecurity than many of us would have liked, I think it struck a reasonable balance between new regulations (almost none) and guidance (in the form of nonbinding recommendations).")
	suite.Nil(item.Errors(), "There should be no errors at the item level")

	descr, nlpErr := item.Summary().FirstSentenceOfBody()
	suite.Nil(nlpErr, "Unable to retrieve first sentence of item.Summary(): %v", nlpErr)
	suite.Equal(descr, "In 2013, the Food and Drug Administration (FDA) issued its first cybersecurity safety communication, followed in 2014 by final guidance.")

	item = collection.Items[2]
	suite.Equal(item.Title().Original(), "Signify Research HIMSS 2019 Show Report (PDF)")
	suite.Nil(item.Errors(), "There should be no errors at the item level")
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(DropmarkSuite))
}

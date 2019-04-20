package dropmark

type IssueCode string

const (
	UnableToCreateHTTPRequest        IssueCode = "DROPMARKAPIE-0100"
	UnableToExecuteHTTPGETRequest    IssueCode = "DROPMARKAPIE-0200"
	InvalidAPIRespHTTPStatusCode     IssueCode = "DROPMARKAPIE-0300"
	UnableToReadBodyFromHTTPResponse IssueCode = "DROPMARKAPIE-0400"
)

// Issue is a structured problem identification with context information
type Issue interface {
	IssueContext() interface{} // this will be the Dropmark API, it's kept generic so it doesn't require package dependency
	IssueCode() IssueCode      // useful to uniquely identify a particular code
	Issue() string             // the

	IsError() bool   // this issue is an error
	IsWarning() bool // this issue is a warning
}

// Issues packages multiple issues into a container
type Issues interface {
	Issues() []Issue
	IssueCounts() (uint, uint, uint)
	HandleIssues(errorHandler func(Issue), warningHandler func(Issue))
}

type issue struct {
	apiEndpoint string
	code        IssueCode
	message     string
	isError     bool
	children    []Issue
}

func newIssue(apiEndpoint string, code IssueCode, message string, isError bool) Issue {
	result := new(issue)
	result.apiEndpoint = apiEndpoint
	result.code = code
	result.message = message
	result.isError = isError
	return result
}

func (i issue) IssueContext() interface{} {
	return i.apiEndpoint
}

func (i issue) IssueCode() IssueCode {
	return i.code
}

func (i issue) Issue() string {
	return i.message
}

func (i issue) IsError() bool {
	return i.isError
}

func (i issue) IsWarning() bool {
	return !i.isError
}

// Error satisfies the Go error contract
func (i issue) Error() string {
	return i.message
}

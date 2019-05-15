package dropmark

import (
	"io"
)

// ReaderProgressReporter is sent to this package's methods if activity progress reporting is expected
type ReaderProgressReporter interface {
	StartReportableReaderActivityInBytes(summary string, exepectedBytes int64, inputReader io.Reader) io.Reader
	CompleteReportableActivityProgress(summary string)
}

var defaultProgressReporter = &silentProgressReporter{}

type silentProgressReporter struct{}

func (pr *silentProgressReporter) StartReportableReaderActivityInBytes(summary string, exepectedBytes int64, inputReader io.Reader) io.Reader {
	return inputReader
}

func (pr *silentProgressReporter) CompleteReportableActivityProgress(summary string) {

}

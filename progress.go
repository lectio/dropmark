package dropmark

import (
	"context"
	"fmt"
	"io"
)

// ReaderProgressReporter is sent to this package's methods if activity progress reporting is expected for an io.Reader
type ReaderProgressReporter interface {
	StartReportableReaderActivityInBytes(ctx context.Context, summary string, exepectedBytes int64, inputReader io.Reader) io.Reader
	CompleteReportableActivityProgress(ctx context.Context, summary string)
}

// BoundedProgressReporter is one observation method for live reporting of long-running processes where the upper bound is known
type BoundedProgressReporter interface {
	StartReportableActivity(ctx context.Context, summary string, expectedItems int)
	IncrementReportableActivityProgress(ctx context.Context)
	IncrementReportableActivityProgressBy(ctx context.Context, incrementBy int)
	CompleteReportableActivityProgress(ctx context.Context, summary string)
}

var defaultProgressReporter = silentProgressReporter{}

type silentProgressReporter struct{}
type summaryProgressReporter struct{ prefix string }

func (pr silentProgressReporter) StartReportableActivity(ctx context.Context, summary string, expectedItems int) {
}

func (pr silentProgressReporter) StartReportableReaderActivityInBytes(ctx context.Context, summary string, exepectedBytes int64, inputReader io.Reader) io.Reader {
	return inputReader
}

func (pr silentProgressReporter) IncrementReportableActivityProgress(ctx context.Context) {
}

func (pr silentProgressReporter) IncrementReportableActivityProgressBy(ctx context.Context, incrementBy int) {
}

func (pr silentProgressReporter) CompleteReportableActivityProgress(ctx context.Context, summary string) {
}

func (pr summaryProgressReporter) StartReportableActivity(ctx context.Context, summary string, expectedItems int) {
	fmt.Printf("[%s] %s\n", pr.prefix, summary)
}

func (pr summaryProgressReporter) StartReportableReaderActivityInBytes(ctx context.Context, summary string, exepectedBytes int64, inputReader io.Reader) io.Reader {
	fmt.Printf("[%s] %s\n", pr.prefix, summary)
	return inputReader
}

func (pr summaryProgressReporter) IncrementReportableActivityProgress(ctx context.Context) {
}

func (pr summaryProgressReporter) IncrementReportableActivityProgressBy(ctx context.Context, incrementBy int) {
}

func (pr summaryProgressReporter) CompleteReportableActivityProgress(ctx context.Context, summary string) {
	fmt.Printf("[%s] %s\n", pr.prefix, summary)
}

package jobs

import (
	"context"

	"korus/internal/scanner"
)

type ScannerAdapter struct {
	scanner *scanner.Scanner
}

func NewScannerAdapter(s *scanner.Scanner) *ScannerAdapter {
	return &ScannerAdapter{scanner: s}
}

func (a *ScannerAdapter) ScanLibrary(ctx context.Context, force bool) (*ScanResult, error) {
	result, err := a.scanner.ScanLibrary(ctx, force)
	if err != nil {
		return nil, err
	}

	// Convert scanner.ScanResult to jobs.ScanResult
	return &ScanResult{
		FilesFound:    result.FilesFound,
		FilesAdded:    result.FilesAdded,
		FilesUpdated:  result.FilesUpdated,
		FilesRemoved:  result.FilesRemoved,
		Duration:      result.Duration,
		Errors:        result.Errors,
	}, nil
}
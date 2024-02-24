package speeches

import "context"

type (
	Transcriber interface {
		Transcribe(ctx context.Context, filePath string) (TranscribeResult, error)
	}

	TranscribeResult struct {
		Segments []TranscribeSegmentResult
	}

	TranscribeSegmentResult struct {
		ID      uint64
		StartMs uint64
		EndMs   uint64
		Text    string
		Words   []TranscribeWordResult
	}

	TranscribeWordResult struct {
		ID      uint64
		Text    string
		StartMs *uint64
		EndMs   *uint64
	}
)

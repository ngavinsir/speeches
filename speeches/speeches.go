package speeches

import "io"

type (
	Speech struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Blake3Hash    string `json:"blake3_hash"`
		IsTranscribed bool   `json:"is_transcribed"`
	}

	Word struct {
		SpeechID string `json:"speech_id"`
		Text     string `json:"text"`
		StartMs  uint64 `json:"start_ms"`
		EndMs    uint64 `json:"end_ms"`
	}

	CreateSpeechReq struct {
		Name string
		File io.Reader
	}
)

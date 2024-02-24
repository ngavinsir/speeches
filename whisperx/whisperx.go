package whisperx

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"speeches/speeches"

	"github.com/shopspring/decimal"
)

type (
	transcribeResult struct {
		Segments []segment `json:"segments"`
	}

	segment struct {
		Text  string          `json:"text"`
		Start decimal.Decimal `json:"start"`
		End   decimal.Decimal `json:"end"`
		Words []word          `json:"words"`
	}

	word struct {
		Text  string           `json:"word"`
		Start *decimal.Decimal `json:"start"`
		End   *decimal.Decimal `json:"end"`
	}
)

type WhisperxTranscriber struct{}

var _ speeches.Transcriber = WhisperxTranscriber{}

func (w WhisperxTranscriber) Transcribe(ctx context.Context, filePath string) (speeches.TranscribeResult, error) {
	cmd := exec.CommandContext(ctx, "whisperx", filePath)

	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Start()

	go func() {
		scanner := bufio.NewScanner(stderr)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			m := scanner.Text()
			log.Println(m)
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			m := scanner.Text()
			log.Println(m)
		}
	}()

	err := cmd.Wait()
	if err != nil {
		return speeches.TranscribeResult{}, fmt.Errorf("transcribing with whisperx: %w", err)
	}

	filePathWithoutExt := filePath[:len(filePath)-len(filepath.Ext(filePath))]
	transcribeResultPath := filePathWithoutExt + ".json"
	transcribeResultJSONFile, err := os.Open(transcribeResultPath)
	if err != nil {
		return speeches.TranscribeResult{}, fmt.Errorf("opening whisperx transcribe result: %w", err)
	}

	var tr transcribeResult
	err = json.NewDecoder(transcribeResultJSONFile).Decode(&tr)
	if err != nil {
		return speeches.TranscribeResult{}, fmt.Errorf("decoding whisperx json result: %w", err)
	}

	res := speeches.TranscribeResult{
		Segments: make([]speeches.TranscribeSegmentResult, len(tr.Segments)),
	}
	for n, s := range tr.Segments {
		res.Segments[n] = speeches.TranscribeSegmentResult{
			Text:    s.Text,
			StartMs: s.Start.Mul(decimal.NewFromInt(1000)).BigInt().Uint64(),
			EndMs:   s.End.Mul(decimal.NewFromInt(1000)).BigInt().Uint64(),
			Words:   transcribeWorldResultsFromWords(s.Words),
			ID:      uint64(n),
		}
	}
	return res, nil
}

func transcribeWorldResultsFromWords(words []word) []speeches.TranscribeWordResult {
	res := make([]speeches.TranscribeWordResult, len(words))
	for n, w := range words {
		var startMs, endMs *uint64
		if w.Start != nil {
			s := w.Start.Mul(decimal.NewFromInt(1000)).BigInt().Uint64()
			startMs = &s
		}
		if w.End != nil {
			e := w.End.Mul(decimal.NewFromInt(1000)).BigInt().Uint64()
			endMs = &e
		}

		res[n] = speeches.TranscribeWordResult{
			ID:      uint64(n),
			Text:    w.Text,
			StartMs: startMs,
			EndMs:   endMs,
		}
	}
	return res
}

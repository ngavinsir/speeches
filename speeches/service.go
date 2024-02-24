package speeches

import (
	"context"
	"fmt"
	"log"
	"os"
	"speeches/b3"
	"sync"
)

type (
	repo interface {
		CreateSpeech(ctx context.Context, name string, blake3Hash string) (Speech, error)
		InsertSegments(ctx context.Context, t TranscribeResult, speechID string) error
		GetSpeechByHash(ctx context.Context, blake3Hash string) (Speech, error)
	}

	svcImpl struct {
		r  repo
		t  Transcriber
		wg *sync.WaitGroup
	}
)

func NewService(r repo, t Transcriber) svcImpl {
	var wg sync.WaitGroup
	return svcImpl{r: r, t: t, wg: &wg}
}

func (s svcImpl) Wait() {
	s.wg.Wait()
}

func (s svcImpl) StartTranscribe(ctx context.Context, name string, filePath string) (Speech, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Speech{}, fmt.Errorf("start transcribe: opening file: %w", err)
	}

	blake3Hash, err := b3.Blake3HashFromFile(file)
	if err != nil {
		return Speech{}, fmt.Errorf("start transcribe: %w", err)
	}

	speech, err := s.r.CreateSpeech(ctx, name, blake3Hash)
	if err != nil {
		return Speech{}, fmt.Errorf("start transcribe: %w", err)
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		return speech, fmt.Errorf("start transcribe: reset file: %w", err)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		tr, err := s.t.Transcribe(context.Background(), filePath)
		if err != nil {
			log.Println("transcribing: %w", err)
		}

		err = s.r.InsertSegments(context.Background(), tr, speech.ID)
		if err != nil {
			log.Println("transcribing: %w", err)
		}
	}()

	return speech, nil
}

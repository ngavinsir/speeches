package main

import (
	"context"
	"fmt"
	"log"
	"speeches/speeches"
	"speeches/whisperx"
)

func main() {
	// runServer()
	db := initDB()
	s := speeches.NewService(speeches.NewSQLiteRepo(db), whisperx.WhisperxTranscriber{})
	speech, err := s.StartTranscribe(context.Background(), "phmch1", "./phm_ch1.mp3")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("speech: %+v\n", speech)
	s.Wait()
}

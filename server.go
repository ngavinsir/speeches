package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"speeches/speeches"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func runServer() {
	db := initDB()
	defer db.Close()

	mux := http.NewServeMux()

	srv := &http.Server{}
	srv.Addr = ":8121"
	srv.Handler = mux

	r := speeches.NewSQLiteRepo(db)
	s, err := r.CreateSpeech(context.Background(), "test", time.Now().Format(time.RFC3339))
	fmt.Printf("s: %+v, err: %+v\n", s, err)

	// va.InstallController(mux, va.NewRepo(db))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http listen and serve: %v\n", err)
		}
	}()

	<-ctx.Done()
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Printf("shutdown server: %v\n", err)
	}
}

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "file:./speeches.db")
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`
	PRAGMA busy_timeout       = 10000;
	PRAGMA journal_mode       = WAL;
	PRAGMA journal_size_limit = 200000000;
	PRAGMA synchronous        = NORMAL;
	PRAGMA foreign_keys       = ON;
	PRAGMA temp_store         = MEMORY;
	PRAGMA cache_size         = -16000;

	create table if not exists speeches ( 
		id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, 
		name text not null, 
		blake3_hash text not null unique,
		is_transcribed integer default 0
	);
	
	create table if not exists segments ( 
		id integer not null,
		speech_id integer not null,
		text text not null, 
		start_ms integer not null,
		end_ms integer not null,
		primary key (id, speech_id)
	);

	create table if not exists words ( 
		id integer not null,
		segment_id integer not null,
		speech_id integer not null,
		text text not null, 
		start_ms integer,
		end_ms integer,
		primary key (id, segment_id, speech_id)
	);`)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

package speeches

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
)

type (
	SQLiteRepo struct {
		db *sql.DB
	}
)

func NewSQLiteRepo(db *sql.DB) SQLiteRepo {
	return SQLiteRepo{db}
}

func (r SQLiteRepo) GetSpeechByHash(ctx context.Context, blake3Hash string) (Speech, error) {
	res := Speech{}

	err := r.db.
		QueryRowContext(
			ctx,
			"select id, name, blake3_hash from speeches where blake3_hash = $1",
			blake3Hash,
		).
		Scan(&res.ID, &res.Name, &res.Blake3Hash)
	if err != nil {
		return res, fmt.Errorf("get speech by hash: %w", err)
	}

	return res, nil
}

func (r SQLiteRepo) CreateSpeech(ctx context.Context, name string, blake3Hash string) (Speech, error) {
	var isTranscribed uint8
	res := Speech{
		Name:          name,
		Blake3Hash:    blake3Hash,
		IsTranscribed: false,
	}

	err := r.db.
		QueryRowContext(
			ctx,
			"insert into speeches (name, blake3_hash) values ($1, $2) on conflict do nothing returning id, is_transcribed",
			name,
			blake3Hash,
		).
		Scan(&res.ID, &isTranscribed)
	if err != nil {
		return res, fmt.Errorf("persisting speech into sqlite: %w", err)
	}

	if isTranscribed == 1 {
		res.IsTranscribed = true
	}

	return res, nil
}

func (r SQLiteRepo) InsertSegments(ctx context.Context, t TranscribeResult, speechID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("inserting segments: begin trx: %w", err)
	}

	err = r.insertSegments(ctx, tx, t, speechID)
	if err != nil {
		if err = tx.Rollback(); err != nil {
			return fmt.Errorf("rollback insert segments: %w", err)
		}
		return err
	}
	err = r.insertWords(ctx, tx, t, speechID)
	if err != nil {
		if err = tx.Rollback(); err != nil {
			return fmt.Errorf("rollback insert words: %w", err)
		}
		return err
	}
	_, err = tx.ExecContext(ctx, `
		update speeches
		set is_transcribed = 1
		where id = $1
	`, speechID)
	if err != nil {
		if err = tx.Rollback(); err != nil {
			return fmt.Errorf("rollback update speech: %w", err)
		}
		return fmt.Errorf("updating speech is_transcribed: %w", err)
	}

	log.Println("commit")
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("inserting segments: commiting: %w", err)
	}
	log.Println("commit done")
	return nil
}

func (r SQLiteRepo) insertSegments(ctx context.Context, tx *sql.Tx, t TranscribeResult, speechID string) error {
	var insertSegmentsQueryBuilder strings.Builder
	_, err := insertSegmentsQueryBuilder.WriteString(`insert into segments (
		id,
		speech_id,
		text,
		start_ms,
		end_ms) values `)
	if err != nil {
		return fmt.Errorf("inserting segments: building insert segments query: %w", err)
	}
	insertSegmentsArgs := make([]any, 5*len(t.Segments))
	for n, s := range t.Segments {
		prefix := ", "
		if n == 0 {
			prefix = ""
		}
		suffix := ""
		if n == len(t.Segments)-1 {
			suffix = ";"
		}
		b := n * 5
		_, err = insertSegmentsQueryBuilder.WriteString(fmt.Sprintf(`%s(
			$%d, $%d, $%d, $%d, $%d
		)%s`, prefix, b+1, b+2, b+3, b+4, b+5, suffix))
		if err != nil {
			return fmt.Errorf("inserting segments: building insert segments query: %w", err)
		}

		insertSegmentsArgs[b] = s.ID
		insertSegmentsArgs[b+1] = speechID
		insertSegmentsArgs[b+2] = s.Text
		insertSegmentsArgs[b+3] = s.StartMs
		insertSegmentsArgs[b+4] = s.EndMs
	}

	log.Println(insertSegmentsQueryBuilder.String())
	_, err = tx.ExecContext(ctx, insertSegmentsQueryBuilder.String(), insertSegmentsArgs...)
	if err != nil {
		return fmt.Errorf("inserting segments: %w", err)
	}

	return nil
}

func (r SQLiteRepo) insertWords(ctx context.Context, tx *sql.Tx, t TranscribeResult, speechID string) error {
	totalWordsCount := 0
	for _, s := range t.Segments {
		totalWordsCount += len(s.Words)
	}

	var insertWordsQueryBuilder strings.Builder
	_, err := insertWordsQueryBuilder.WriteString(`insert into words (
		id,
		segment_id,
		speech_id,
		text,
		start_ms,
		end_ms) values `)
	if err != nil {
		return fmt.Errorf("inserting words: building insert words query: %w", err)
	}
	insertWordsArgs := make([]any, 6*totalWordsCount)
	b := 0
	for i, s := range t.Segments {
		for j, w := range s.Words {
			prefix := ", "
			if i == 0 && j == 0 {
				prefix = ""
			}
			suffix := ""
			if i == len(t.Segments)-1 && j == len(s.Words)-1 {
				suffix = ";"
			}
			_, err = insertWordsQueryBuilder.WriteString(fmt.Sprintf(`%s(
				$%d, $%d, $%d, $%d, $%d, $%d
			)%s`, prefix, b+1, b+2, b+3, b+4, b+5, b+6, suffix))
			if err != nil {
				return fmt.Errorf("inserting segments: building insert segments query: %w", err)
			}

			insertWordsArgs[b] = w.ID
			insertWordsArgs[b+1] = s.ID
			insertWordsArgs[b+2] = speechID
			insertWordsArgs[b+3] = s.Text
			insertWordsArgs[b+4] = s.StartMs
			insertWordsArgs[b+5] = s.EndMs
			b += 6
		}
	}

	log.Println(insertWordsQueryBuilder.String())
	_, err = tx.ExecContext(ctx, insertWordsQueryBuilder.String(), insertWordsArgs...)
	if err != nil {
		return fmt.Errorf("inserting segments: %w", err)
	}

	return nil
}

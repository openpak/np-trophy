package main

import (
	"context"
	"openpak/np-trophy/native"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Unlock is one earned trophy. IDs are the firmware's zero-based trophy index, EarnedAt is
// Unix seconds and Grade is "", "B", "S", "G" or "P".
type Unlock = native.Unlock

// Game is the persisted native trophy set.
type Game = native.TrophyGame

type Store struct{ DB *pgxpool.Pool }

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, `CREATE TABLE IF NOT EXISTS trophy_unlocks (
		account          uuid    NOT NULL,
		communication_id text    NOT NULL,
		trophy_id        integer NOT NULL,
		earned_at        bigint  NOT NULL,
		grade            text    NOT NULL DEFAULT '',
		PRIMARY KEY (account, communication_id, trophy_id))`)
	return err
}

// Merge records unlocks. The first stored unlock of a trophy is kept (a later sync never
// moves its time), and a grade only fills in where none was recorded yet.
func (s *Store) Merge(ctx context.Context, account, communicationID string, unlocks []Unlock) error {
	batch := &pgx.Batch{}
	for _, u := range unlocks {
		batch.Queue(`INSERT INTO trophy_unlocks (account, communication_id, trophy_id, earned_at, grade)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (account, communication_id, trophy_id) DO UPDATE
			SET grade = CASE WHEN trophy_unlocks.grade = '' THEN EXCLUDED.grade ELSE trophy_unlocks.grade END`,
			account, communicationID, u.ID, u.EarnedAt, u.Grade)
	}
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Games returns every trophy set the account has unlocks in, by communication ID and trophy ID.
func (s *Store) Games(ctx context.Context, account string) ([]Game, error) {
	rows, err := s.DB.Query(ctx, `SELECT communication_id, trophy_id, earned_at, grade FROM trophy_unlocks
		WHERE account = $1 ORDER BY communication_id, trophy_id`, account)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	games := []Game{}
	for rows.Next() {
		var comm string
		var u Unlock
		if err := rows.Scan(&comm, &u.ID, &u.EarnedAt, &u.Grade); err != nil {
			return nil, err
		}
		if n := len(games); n == 0 || games[n-1].CommunicationID != comm {
			games = append(games, Game{CommunicationID: comm})
		}
		games[len(games)-1].Trophies = append(games[len(games)-1].Trophies, u)
	}
	return games, rows.Err()
}

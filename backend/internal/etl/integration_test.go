package etl

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func integrationDB(t *testing.T, env string) *pgx.Conn {
	t.Helper()
	dsn := os.Getenv(env)
	if dsn == "" {
		t.Skipf("%s not set; skipping live-database ETL test", env)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect using %s: %v", env, err)
	}
	t.Cleanup(func() { _ = conn.Close(context.Background()) })
	return conn
}

func TestIntegrationCoreSchema(t *testing.T) {
	db := integrationDB(t, "TEST_ETL_DATABASE_URL")
	const query = `SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name IN (
			'data_set','generation','type','stat','nature','species',
			'move','move_meta','move_stat_change','ability','item')`
	var count int
	if err := db.QueryRow(context.Background(), query).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 11 {
		t.Errorf("core tables = %d, want 11", count)
	}
}

func TestIntegrationCoreGrants(t *testing.T) {
	ctx := context.Background()
	etl := integrationDB(t, "TEST_ETL_DATABASE_URL")
	app := integrationDB(t, "TEST_DATABASE_URL")

	tx, err := etl.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback also runs after a failed insert
	if _, err := tx.Exec(ctx, `INSERT INTO data_set (identifier) VALUES ('ad6-probe')`); err != nil {
		t.Fatalf("ETL role cannot write data_set: %v", err)
	}

	var count int
	if err := app.QueryRow(ctx, `SELECT count(*) FROM data_set`).Scan(&count); err != nil {
		t.Fatalf("app role cannot read data_set: %v", err)
	}
	_, err = app.Exec(ctx, `INSERT INTO data_set (identifier) VALUES ('ad6-violation')`)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Errorf("app role INSERT error = %v, want insufficient_privilege (42501)", err)
	}
}

func TestIntegrationLoadedData(t *testing.T) {
	db := integrationDB(t, "TEST_DATABASE_URL")
	ctx := context.Background()
	tests := []struct {
		name, query, want string
	}{
		{"species count", `SELECT count(*)::text FROM species`, "1351"},
		{"gen-9 move count", `SELECT count(*)::text FROM move m JOIN data_set d ON d.id = m.data_set_id WHERE d.identifier = 'gen-9'`, "919"},
		{"gen-9 swords-dance effect", `SELECT m.identifier FROM move_stat_change msc
			JOIN move m ON m.id = msc.move_id
			JOIN data_set d ON d.id = m.data_set_id
			JOIN stat s ON s.id = msc.stat_id
			WHERE s.identifier = 'attack' AND msc.change = 2
			AND d.identifier = 'gen-9' AND m.identifier = 'swords-dance'`, "swords-dance"},
		{"champions roster", `SELECT p.identifier || ' ' || (SELECT count(*) FROM move WHERE data_set_id = d.id)
			|| ' ' || (SELECT count(*) FROM ability WHERE data_set_id = d.id)
			|| ' ' || (SELECT count(*) FROM item WHERE data_set_id = d.id)
			FROM data_set d JOIN data_set p ON p.id = d.parent_id
			WHERE d.identifier = 'champions'`, "gen-9 497 200 148"},
		{"champions beak-blast", `SELECT m.power || '/' || m.pp FROM move m JOIN data_set d ON d.id = m.data_set_id
			WHERE d.identifier = 'champions' AND m.identifier = 'beak-blast'`, "120/8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			if err := db.QueryRow(ctx, tt.query).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

package core

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/pressly/goose/v3"
)

type BaseFixture struct {
	ctx      context.Context
	db       *sql.DB
	t        *testing.T
	tearDown func()
}

func NewBaseFixture(t *testing.T) *BaseFixture {

	ctx, cancel := context.WithCancel(context.Background())

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}

	migrationfs := os.DirFS("../migrations")
	goose.SetBaseFS(migrationfs)

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}

	if err := goose.Up(db, "."); err != nil {
		t.Fatal(err)
	}

	return &BaseFixture{
		ctx: ctx,
		db:  db,
		t:   t,
		tearDown: func() {
			cancel()
			db.Close()
		},
	}
}

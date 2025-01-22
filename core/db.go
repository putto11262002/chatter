package core

import (
	"database/sql"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

type SQLiteDBOption struct {
	// mode can be ro | rw | rwc | memory
	Mode string
	// cache can be shared | private
	Cache string
	// JournalMode be DELETE | TRUNCATE | PERSIST | MEMORY | WAL | OFF
	JournalMode string
}

func (config *SQLiteDBOption) DSN(sb *strings.Builder) {
	if config == nil {
		return
	}

	if config.Mode != "" {
		sb.WriteString("?mode=")
		sb.WriteString(config.Mode)
	}

	if config.Cache != "" {
		sb.WriteString("&cache=")
		sb.WriteString(config.Cache)
	}

	if config.JournalMode != "" {
		sb.WriteString("&journal_mode=")
		sb.WriteString(config.JournalMode)
	}

}

type SQLiteDB struct {
	*sql.DB
	config       *SQLiteDBOption
	file         string
	migrationDir string
}

func NewSQLiteDB(file, migrationDir string, config *SQLiteDBOption) (*SQLiteDB, error) {
	db := &SQLiteDB{config: config, migrationDir: migrationDir, file: file}

	var dsn strings.Builder
	dsn.WriteString("file:")
	dsn.WriteString(db.file)

	if db.config != nil {
		config.DSN(&dsn)
	}
	d, err := sql.Open("sqlite3", dsn.String())
	if err != nil {
		return nil, err
	}

	db.DB = d
	return db, nil
}

func (db *SQLiteDB) Migrate() error {
	migrationfs := os.DirFS(db.migrationDir)
	goose.SetBaseFS(migrationfs)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err

	}

	if err := goose.Up(db.DB, "."); err != nil {
		return err
	}
	return nil
}

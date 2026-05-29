// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wavetermdev/waveterm/pkg/mcstore"
	"github.com/wavetermdev/waveterm/pkg/util/migrateutil"

	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
)

func initDB(dataHome string) error {
	if err := os.MkdirAll(dataHome, 0700); err != nil {
		return fmt.Errorf("creating MC_DATA_HOME %q: %w", dataHome, err)
	}
	dbPath := filepath.Join(dataHome, "mc.db")
	db, err := sqlx.Open("sqlite3", fmt.Sprintf(
		"file:%s?mode=rwc&_journal_mode=WAL&_busy_timeout=5000", dbPath,
	))
	if err != nil {
		return fmt.Errorf("opening sqlite db: %w", err)
	}
	db.SetMaxOpenConns(1)

	err = migrateutil.Migrate("mcstore", db.DB, mcstore.MigrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("running mcstore migrations: %w", err)
	}
	mcstore.SetDB(db)
	log.Printf("mcstore initialized: %s\n", dbPath)
	return nil
}

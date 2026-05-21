package main

import (
	"log"
	"os"

	generalConf "github.com/flamefks/scheduler-system/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")

	m, err := migrate.New(
		"file:///app/sql/migrations",
		dsn,
	)
	if err != nil {
		log.Fatalf("migrate init failed: %s", generalConf.RedactText(err.Error()))
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration failed: %s", generalConf.RedactText(err.Error()))
	}

	log.Println("migrations applied")
}

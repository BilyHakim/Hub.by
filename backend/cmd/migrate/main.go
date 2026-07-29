package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/hubby-id/hubby/backend/internal/config"
	"github.com/hubby-id/hubby/backend/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	cfg := config.Load()
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("connect database: %v", err)
	}

	goose.SetBaseFS(migrations.Files)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("set goose dialect: %v", err)
	}

	if err := run(db, command); err != nil {
		log.Fatalf("goose %s: %v", command, err)
	}
}

func run(db *sql.DB, command string) error {
	switch command {
	case "up":
		return goose.Up(db, ".")
	case "up-by-one":
		return goose.UpByOne(db, ".")
	case "down":
		return goose.Down(db, ".")
	case "status":
		return goose.Status(db, ".")
	case "version":
		version, err := goose.GetDBVersion(db)
		if err == nil {
			fmt.Printf("database migration version: %d\n", version)
		}
		return err
	default:
		return fmt.Errorf("unknown command %q; use up, up-by-one, down, status, or version", command)
	}
}

package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	// ✅ PURE GO SQLITE DRIVER (Railway compatible)
	_ "modernc.org/sqlite"
)

const dbPath = "./app.db"

type AppDB struct {
	DB *sql.DB
}

func NewAppDB() (*AppDB, error) {
	db, err := Setup()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func DropDatabase() error {
	if dbPath == "" || dbPath == ":memory:" {
		return fmt.Errorf("refusing to drop in-memory database")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	db.Close()

	err = os.Remove(dbPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove database file: %w", err)
	}

	return nil
}

func Setup() (*AppDB, error) {

	// ✅ OPEN DB (no CGO required)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping error: %w", err)
	}

	// ===== MIGRATIONS =====
	migrateDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open migration db: %w", err)
	}
	defer migrateDB.Close()

	driver, err := sqlite3.WithInstance(migrateDB, &sqlite3.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"sqlite3",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("migrate.New error: %w", err)
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("migration error: %w", err)
	}

	m.Close()

	log.Println("✅ Migrations applied successfully")

	return &AppDB{DB: db}, nil
}

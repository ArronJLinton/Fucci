//go:build migrate
// +build migrate

package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ArronJLinton/fucci-api/internal/config"
	"github.com/lib/pq"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
)

type Migration struct {
	Version string
	Name    string
	Up      string
	Down    string
}

func main() {
	// Initialize logger
	zlog, _ := zap.NewProduction()
	defer func() {
		_ = zlog.Sync()
	}()
	logger := otelzap.New(zlog)

	// Initialize config
	cfg := config.InitConfig(logger)
	dbURL := normalizePostgresURL(os.Getenv("DB_URL"), cfg.DB_URL)
	if dbURL == "" {
		log.Fatal("DB_URL is required (set DB_URL env or db_url in .env)")
	}
	if !strings.HasPrefix(dbURL, "postgres://") && !strings.HasPrefix(dbURL, "postgresql://") {
		log.Fatal("DB_URL must start with postgres:// or postgresql:// (direct Postgres connection string, not an HTTP API URL)")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v\n"+
			"Hint: use a single-line postgres URL; URL-encode special characters in the password; "+
			"remove accidental quotes or line breaks from GitHub/Fly secrets.", err)
	}

	fmt.Println("Connected to database successfully")

	// Create migrations table if it doesn't exist
	createMigrationsTable(db)

	// Get all migration files
	migrationFiles, err := getMigrationFiles("sql/schema")
	if err != nil {
		log.Fatal("Failed to get migration files:", err)
	}

	// Get already executed migrations
	executedMigrations, err := getExecutedMigrations(db)
	if err != nil {
		log.Fatal("Failed to get executed migrations:", err)
	}

	// Run pending migrations
	pendingMigrations := getPendingMigrations(migrationFiles, executedMigrations)

	if len(pendingMigrations) == 0 {
		fmt.Println("No pending migrations found")
		return
	}

	fmt.Printf("Found %d pending migrations\n", len(pendingMigrations))

	for _, migration := range pendingMigrations {
		fmt.Printf("Running migration: %s\n", migration.Name)

		// Read migration file
		content, err := ioutil.ReadFile(filepath.Join("sql/schema", migration.Name))
		if err != nil {
			log.Fatalf("Failed to read migration file %s: %v", migration.Name, err)
		}

		// Extract UP migration (everything before -- +goose Down)
		upMigration := extractUpMigration(string(content))

		// Execute migration
		if _, err := db.Exec(upMigration); err != nil {
			// Check for PostgreSQL error codes indicating object already exists
			if pqErr, ok := err.(*pq.Error); ok {
				switch pqErr.Code {
				case "42P07", // duplicate_table
					"42710", // duplicate_object
					"42723": // duplicate_function
					log.Printf("Warning: Migration %s skipped (object already exists: %s)\n", migration.Name, pqErr.Code)
					// Still record as executed to prevent future attempts
				default:
					log.Fatalf("Migration %s failed: %v (PostgreSQL error code: %s)", migration.Name, err, pqErr.Code)
				}
			} else {
				// Fallback for non-PostgreSQL errors
				log.Fatalf("Migration %s failed: %v", migration.Name, err)
			}
		}

		// Record migration as executed
		if err := recordMigration(db, migration.Version, migration.Name); err != nil {
			log.Fatalf("Failed to record migration %s: %v", migration.Name, err)
		}

		fmt.Printf("Migration %s completed successfully\n", migration.Name)
	}

	fmt.Println("All pending migrations completed successfully!")
}

// normalizePostgresURL trims whitespace and stray quotes often introduced by secret managers and CI.
// Prefer OS env (explicit in GitHub Actions) over viper/config file.
func normalizePostgresURL(fromEnv, fromCfg string) string {
	s := strings.TrimSpace(fromEnv)
	if s == "" {
		s = strings.TrimSpace(fromCfg)
	}
	s = strings.Trim(s, "\r\n\t ")
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			s = strings.TrimSpace(s[1 : len(s)-1])
		}
	}
	return strings.TrimSpace(s)
}

func createMigrationsTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(query); err != nil {
		log.Fatal("Failed to create migrations table:", err)
	}
}

func getMigrationFiles(schemaDir string) ([]Migration, error) {
	files, err := ioutil.ReadDir(schemaDir)
	if err != nil {
		return nil, err
	}

	var migrations []Migration
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".sql") {
			// Extract version from filename (e.g., "001_users.sql" -> "001")
			version := strings.Split(file.Name(), "_")[0]
			migrations = append(migrations, Migration{
				Version: version,
				Name:    file.Name(),
			})
		}
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func getExecutedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	executed := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		executed[version] = true
	}

	return executed, nil
}

func getPendingMigrations(allMigrations []Migration, executed map[string]bool) []Migration {
	var pending []Migration
	for _, migration := range allMigrations {
		if !executed[migration.Version] {
			pending = append(pending, migration)
		}
	}
	return pending
}

func extractUpMigration(content string) string {
	// Find the -- +goose Down marker
	downMarker := "-- +goose Down"
	downIndex := strings.Index(content, downMarker)

	if downIndex == -1 {
		// No down migration, return everything after -- +goose Up
		upMarker := "-- +goose Up"
		upIndex := strings.Index(content, upMarker)
		if upIndex == -1 {
			return content
		}
		return strings.TrimSpace(content[upIndex+len(upMarker):])
	}

	// Return everything between -- +goose Up and -- +goose Down
	upMarker := "-- +goose Up"
	upIndex := strings.Index(content, upMarker)
	if upIndex == -1 {
		return strings.TrimSpace(content[:downIndex])
	}

	upContent := content[upIndex+len(upMarker) : downIndex]
	return strings.TrimSpace(upContent)
}

func recordMigration(db *sql.DB, version, name string) error {
	query := "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)"
	_, err := db.Exec(query, version, name)
	return err
}

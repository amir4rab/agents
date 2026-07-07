package sqlite

import (
	"database/sql"
	"fmt"
	"math/rand/v2"
	"runtime"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Pragma string

type Config struct {
	Source  string   // Location of the database file
	pragmas []Pragma // List of the database pragmas
}

const (
	fkPragma          Pragma = "_foreign_keys=on"
	journalPragma     Pragma = "_journal_mode=WAL"
	busyTimeoutPragma Pragma = "_busy_timeout=5000"
	syncPragma        Pragma = "_synchronous=NORMAL"
	memoryPragma      Pragma = "mode=memory"
	cachePragma       Pragma = "cache=shared"
)

// NewConfig returns a file based config based on the provided source
func NewConfig(store string) *Config {
	return &Config{
		Source: store,
		pragmas: []Pragma{
			fkPragma,
			journalPragma,
			busyTimeoutPragma,
			syncPragma,
		}}
}

// NewMemConfig returns a randomly generated memory based config
func NewMemConfig() *Config {
	source := fmt.Sprintf("file:instance-%d", rand.Int64())
	return &Config{
		Source:  source,
		pragmas: []Pragma{memoryPragma, cachePragma}}
}

// ToString converts the provided config into a "dataSourceName" string
func (c *Config) ToString() string {
	builder := strings.Builder{}
	builder.WriteString(c.Source)

	for i, p := range c.pragmas {
		if i == 0 {
			builder.WriteRune('?')
		}

		builder.WriteString(string(p))

		if i != len(c.pragmas)-1 {
			builder.WriteRune('&')
		}
	}

	return builder.String()
}

// Conn creates the connection to the database and tries to ping it to verify a sucessful connection.
func Conn(conf *Config) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", conf.ToString())
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	maxConns := runtime.NumCPU()
	db.SetMaxIdleConns(maxConns)
	db.SetMaxOpenConns(maxConns)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	return db, nil
}

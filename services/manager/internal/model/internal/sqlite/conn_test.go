package sqlite_test

import (
	"strings"
	"testing"

	"github.com/nilafzar/agents/services/manager/internal/model/internal/sqlite"
)

func TestConfigBuilder(t *testing.T) {
	conf := sqlite.NewConfig("./db.sqlite")
	dsn := conf.ToString()

	expectedPragmas := "_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000&_synchronous=NORMAL"
	if !strings.Contains(dsn, expectedPragmas) {
		t.Error("The final dsn doesn't include the expected pragmas")
	}
}

func TestMemConfigBuilder(t *testing.T) {
	conf := sqlite.NewMemConfig()
	dsn := conf.ToString()

	expectedPragmas := "mode=memory&cache=shared"
	if !strings.Contains(dsn, expectedPragmas) {
		t.Error("The final dsn doesn't include the expected pragmas")
	}
}

func TestConn(t *testing.T) {
	conf := sqlite.NewMemConfig()
	_, err := sqlite.Conn(conf)
	if err != nil {
		t.Error(err)
	}

}

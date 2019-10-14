// SQLite3 backend for the tilo server.
//
// Each record has two timestamps, "started" and "ended". They are saved as
// Unix time stamps because some arithmetic is performed on them which is
// cumbersome when storing timestamps as strings.
package sqlite3

import (
	"database/sql"
	"github.com/fgahr/tilo/command/query"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server/backend"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"time"
)

const (
	backendName = "sqlite3"
)

func init() {
	s := SQLite{conf: defaultConf()}
	backend.RegisterBackend(&s)
}

type sqliteConf struct {
	dbFile config.Item
}

func defaultConf() sqliteConf {
	// TODO: Log warning on error?
	home, _ := os.UserHomeDir()
	fileDefault := filepath.Join(home, ".config", "tilo", "tilo.db")
	dbFile := config.Item{
		InFile: "db_file",
		InArgs: "db-file",
		InEnv:  "DB_FILE",
		Value:  fileDefault,
	}
	return sqliteConf{dbFile: dbFile}
}

func (c *sqliteConf) BackendName() string {
	return backendName
}

func (c *sqliteConf) AcceptedItems() []*config.Item {
	return []*config.Item{&c.dbFile}
}

type SQLite struct {
	conf sqliteConf
	db   *sql.DB
}

func (s *SQLite) Config() config.BackendConfig {
	return &s.conf
}

func (s *SQLite) Name() string {
	return backendName
}

func (s *SQLite) Init() error {
	if s == nil {
		return errors.New("No backend present")
	}
	db, err := sql.Open("sqlite3", s.conf.dbFile.Value)
	if err != nil {
		return errors.Wrap(err, "Unable to establish database connection")
	}
	s.db = db
	// Setup schema
	_, err = s.db.Exec(`
CREATE TABLE IF NOT EXISTS task (
	name TEXT NOT NULL,
	started INTEGER NOT NULL,
	ended INTEGER NOT NULL);`)
	if err != nil {
		return errors.Wrap(err, "Unable to setup database")
	}

	_, err = s.db.Exec(
		"CREATE INDEX IF NOT EXISTS task_name ON task (name);")
	return errors.Wrap(err, "Unable to setup database")
}

func (s *SQLite) Close() error {
	if s == nil {
		return errors.New("No backend present")
	}
	return s.db.Close()
}

func (s *SQLite) Save(task msg.Task) error {
	if s == nil {
		return errors.New("No backend present")
	}
	if task.IsRunning() {
		panic("Cannot save an active task.")
	}
	_, err := s.db.Exec(
		"INSERT INTO task (name, started, ended) VALUES (?, ?, ?);",
		task.Name, task.Started.Unix(), task.Ended.Unix())
	return errors.Wrapf(err, "Error while saving %v", task)
}

// Query the total time spent on a task between start and end.
func (s *SQLite) GetTaskBetween(task string, start time.Time, end time.Time) ([]msg.Summary, error) {
	if task == query.TskAllTasks {
		return s.GetAllTasksBetween(start, end)
	}
	// FIXME: total is a non-standard function present in SQLite. Making it
	// work with sum() seems preferable. NULL-behaviour needs to be tested.
	rows, err := s.db.Query(`
SELECT total(ended - started), min(started), max(ended) FROM task
WHERE name = ?
  AND started >= ?
  AND ended < ?
GROUP BY name;`,
		task, start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var duration, started, ended int64
	if rows.Next() {
		err = rows.Scan(&duration, &started, &ended)
		if err != nil {
			return nil, err
		}
		return []msg.Summary{msg.Summary{
			Task:  task,
			Total: time.Duration(duration * int64(time.Second/time.Nanosecond)),
			Start: time.Unix(started, 0),
			End:   time.Unix(ended, 0),
		}}, nil
	}

	return nil, rows.Err()
}

// Query the total time spent on all tasks between start and end.
func (s *SQLite) GetAllTasksBetween(start, end time.Time) ([]msg.Summary, error) {
	rows, err := s.db.Query(`
SELECT name, total(ended-started), min(started), max(ended) FROM task
WHERE started >= ?
  AND ended < ?
GROUP BY name;`,
		start.Unix(), end.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []msg.Summary
	for rows.Next() {
		var taskName string
		var duration, started, ended int64
		err = rows.Scan(&taskName, &duration, &started, &ended)
		if err != nil {
			return result, err
		}
		taskSummary := msg.Summary{
			Task:  taskName,
			Total: time.Duration(duration * int64(time.Second/time.Nanosecond)),
			Start: time.Unix(started, 0),
			End:   time.Unix(ended, 0),
		}
		result = append(result, taskSummary)
	}

	return result, rows.Err()
}

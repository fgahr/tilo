// SQLite3 backend for the tilo server.
//
// Each record has two timestamps, "started" and "ended". They are saved as
// Unix time stamps because some arithmetic is performed on them which is
// cumbersome when storing timestamps as strings.
package sqlite3

import (
	"database/sql"
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

// Query the database based on the given query details.
func (s *SQLite) Query(taskName string, param msg.QueryParam) ([]msg.Summary, error) {
	// TODO: Move this function to the handler instead and keep this out of the backend?
	if len(param) < 2 {
		return nil, errors.Errorf("Invalid query parameter: %v", param)
	}

	var sum []msg.Summary
	if s == nil {
		return sum, errors.New("No backend present")
	}
	var err error
	switch param[0] {
	case msg.QryDay:
		start, err := time.Parse("2006-01-02", param[1])
		if err != nil {
			return nil, errors.Wrap(err, "Unable to construct query")
		}
		end := start.AddDate(0, 0, 1)
		sum, err = s.queryTaskBetween(taskName, start, end)
	case msg.QryBetween:
		if len(param) < 3 {
			return nil, errors.Errorf("Invalid query parameter: %v", param)
		}
		start, err := time.Parse("2006-01-02", param[1])
		if err != nil {
			return nil, err
		}
		end, err := time.Parse("2006-01-02", param[2])
		if err != nil {
			return nil, err
		}
		sum, err = s.queryTaskBetween(taskName, start, end)
	case msg.QryMonth:
		start, err := time.Parse("2006-01", param[1])
		if err != nil {
			return nil, errors.Wrap(err, "Unable to construct query")
		}
		end := start.AddDate(0, 1, 0)
		sum, err = s.queryTaskBetween(taskName, start, end)
	case msg.QryYear:
		start, err := time.Parse("2006", param[1])
		if err != nil {
			return nil, errors.Wrap(err, "Unable to construct query")
		}
		end := start.AddDate(1, 0, 0)
		sum, err = s.queryTaskBetween(taskName, start, end)
	}
	if err != nil {
		return nil, errors.Wrap(err, "Error in database query")
	}

	// Setting the details allows to give better output.
	for i, _ := range sum {
		sum[i].Details = param
	}
	return sum, nil
}

// Query the total time spent on a task between start and end.
func (s *SQLite) queryTaskBetween(task string, start time.Time, end time.Time) ([]msg.Summary, error) {
	if task == msg.TskAllTasks {
		return s.queryAllTasksBetween(start, end)
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
func (s *SQLite) queryAllTasksBetween(start, end time.Time) ([]msg.Summary, error) {
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

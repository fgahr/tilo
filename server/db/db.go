// Package db contains all relevant database queries.
// TODO: Table description
package db

import (
	"database/sql"
	"fmt"
	"github.com/freag/tilo/config"
	"github.com/freag/tilo/msg"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

// Type representing a database backend. At this point, only Sqlite is supported.
type Backend struct {
	conf *config.Params
	db   *sql.DB
}

// Create a new backend based on conf.
func NewBackend(conf *config.Params) *Backend {
	return &Backend{conf: conf}
}

func (b *Backend) Init() error {
	db, err := sql.Open("sqlite3", b.conf.DBFile())
	if err != nil {
		return err
	}
	b.db = db
	// Setup schema
	_, err = b.db.Exec(`
CREATE TABLE IF NOT EXISTS task (
	name TEXT NOT NULL,
	started INTEGER NOT NULL,
	ended INTEGER NOT NULL);`)
	if err != nil {
		return err
	}

	_, err = b.db.Exec(
		"CREATE INDEX IF NOT EXISTS task_name ON task (name);")
	return err
}

func (b *Backend) Close() error {
	return b.db.Close()
}

// Save a task to the database, usually after stopping it first.
func (b *Backend) Save(task *msg.Task) error {
	if !task.HasEnded {
		panic("Cannot save an active task.")
	}
	_, err := b.db.Exec(
		"INSERT INTO task (name, started, ended) VALUES (?, ?, ?);",
		task.Name, task.Started.Unix(), task.Ended.Unix())
	return err
}

// Query the database based on the given query details.
func (b *Backend) Query(taskName string, details msg.QueryDetails) ([]msg.Summary, error) {
	if len(details) < 2 {
		return nil, fmt.Errorf("Invalid query details: %v", details)
	}

	var sum []msg.Summary
	var err error
	switch details[0] {
	case msg.QryDay:
		start, err := time.Parse("2006-01-02", details[1])
		if err != nil {
			return nil, err
		}
		end := start.AddDate(0, 0, 1)
		sum, err = b.queryTaskBetween(taskName, start, end)
	case msg.QryBetween:
		if len(details) < 3 {
			return nil, fmt.Errorf("Invalid query details: %v", details)
		}
		start, err := time.Parse("2006-01-02", details[1])
		if err != nil {
			return nil, err
		}
		end, err := time.Parse("2006-01-02", details[2])
		if err != nil {
			return nil, err
		}
		sum, err = b.queryTaskBetween(taskName, start, end)
	case msg.QryMonth:
		start, err := time.Parse("2006-01", details[1])
		if err != nil {
			return nil, err
		}
		end := start.AddDate(0, 1, 0)
		sum, err = b.queryTaskBetween(taskName, start, end)
	case msg.QryYear:
		start, err := time.Parse("2006", details[1])
		if err != nil {
			return nil, err
		}
		end := start.AddDate(1, 0, 0)
		sum, err = b.queryTaskBetween(taskName, start, end)
	}
	if err != nil {
		return nil, err
	}

	// Setting the details allows to give better output.
	for i, _ := range sum {
		sum[i].Details = details
	}
	return sum, nil
}

// Query the total time spent on a task between start and end.
func (b *Backend) queryTaskBetween(task string, start time.Time, end time.Time) ([]msg.Summary, error) {
	if task == msg.TskAllTasks {
		return b.queryAllTasksBetween(start, end)
	}

	rows, err := b.db.Query(`
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
func (b *Backend) queryAllTasksBetween(start, end time.Time) ([]msg.Summary, error) {
	rows, err := b.db.Query(`
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

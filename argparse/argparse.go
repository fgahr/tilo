package argparse

import (
	"strings"
	"github.com/pkg/errors"
	"fmt"
	"os"
)

const (
	allTasks string = "--all"
)

// Warn the user about arguments being unevaluated.
func WarnUnused(args []string) {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "Ignoring unused arguments:", args)
	}
}

// Split task names given as a comma-separated field, check for validity.
func GetTaskNames(taskField string) ([]string, error) {
	if taskField == allTasks {
		return []string{allTasks}, nil
	}

	tasks := strings.Split(taskField, ",")
	for _, task := range tasks {
		if !validTaskName(task) {
			return nil, errors.Errorf("Invalid task name: %s", task)
		}
	}
	return tasks, nil
}

// Whether the given name is valid for a task.
// In particular, task names cannot contain whitespace and cannot start with
// dashes.
func validTaskName(name string) bool {
	if strings.HasPrefix(name, "-") {
		return false
	}

	if strings.ContainsAny(name, " \t\n") {
		return false
	}

	return true
}

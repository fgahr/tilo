package argparse

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"strings"
)

const (
	// FIXME: Shouldn't be a constant here.
	AllTasks string = ":all"
)

// Warn the user about arguments being unevaluated.
func WarnUnused(args []string) {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "Ignoring unused arguments:", args)
	}
}

// Split task names given as a comma-separated field, check for validity.
func GetTaskNames(taskField string) ([]string, error) {
	if taskField == AllTasks {
		return []string{AllTasks}, nil
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
func validTaskName(name string) bool {
	if strings.HasPrefix(name, ":") {
		return false
	} else if strings.ContainsAny(name, " \t\n") {
		return false
	}
	return true
}

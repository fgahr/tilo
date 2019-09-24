package argparse

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"
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
	} else if hasWhitespace(name) {
		return false
	}
	return true
}

func isKeyword(word string) bool {
	return strings.HasPrefix(word, ":") || !hasWhitespace(word)
}

func stripKeyword(raw string) string {
	return strings.TrimLeft(raw, ":")
}

func hasWhitespace(str string) bool {
	return strings.ContainsAny(str, " \t\n")
}

// TODO
type Quantity struct {
	// TODO
}

func (q *Quantity) Add(more Quantity) {
	// TODO
}

type Quantifier interface {
	Parse(str string) (Quantity, error)
	Describe() string
}

type ListQuantifier struct {
	elem Quantifier
}

func ListQuantifierOf(elem Quantifier) Quantifier {
	return ListQuantifier{elem}
}

func (lq ListQuantifier) Parse(str string) (Quantity, error) {
	qnt := Quantity{}
	for _, part := range strings.Split(str, ",") {
		nxt, err := lq.elem.Parse(part)
		if err != nil {
			return qnt, err
		}
		qnt.Add(nxt)
	}
	return qnt, nil
}

func (lq ListQuantifier) Describe() string {
	return fmt.Sprintf("%s,...", lq.elem.Describe())
}

type PairQuantifier struct {
	elem Quantifier
}

func PairQuantifierOf(elem Quantifier) Quantifier {
	return PairQuantifier{elem}
}

func (pq PairQuantifier) Parse(str string) (Quantity, error) {
	qnt := Quantity{}
	fields := strings.Split(str, ":")
	if len(fields) != 2 {
		return qnt, errors.New("Not a pair: " + str)
	}
	for _, part := range fields {
		nxt, err := pq.elem.Parse(part)
		if err != nil {
			return qnt, err
		}
		qnt.Add(nxt)
	}
	return qnt, nil
}

func (pq PairQuantifier) Describe() string {
	return fmt.Sprintf("%s:%[1]s", pq.elem.Describe())
}

type DateQuantifier struct{}

func (dq DateQuantifier) Parse(str string) (Quantity, error) {
	// TODO: Fix once Quantity is defined
	_, err := time.Parse("2006-01-02", str)
	return Quantity{}, err
}

func (dq DateQuantifier) Describe() string {
	return "YYYY-MM-DD"
}

type MonthQuantifier struct{}

func (mq MonthQuantifier) Parse(str string) (Quantity, error) {
	// TODO: Fix once Quantity is defined
	_, err := time.Parse("2006-01", str)
	return Quantity{}, err
}

func (mq MonthQuantifier) Describe() string {
	return "YYYY-MM"
}

type YearQuantifier struct{}

func (yq YearQuantifier) Parse(str string) (Quantity, error) {
	// TODO: Fix once Quantity is defined
	_, err := time.Parse("2006", str)
	return Quantity{}, err
}

func (yq YearQuantifier) Describe() string {
	return "YYYY"
}

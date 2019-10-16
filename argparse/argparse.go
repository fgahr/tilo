package argparse

import (
	"fmt"
	"github.com/fgahr/tilo/msg"
	"github.com/pkg/errors"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ParamIdentifierPrefix = ":"
	// TODO: Should it be a public constant here? Other options? Package-private?
	AllTasks string = ParamIdentifierPrefix + "all"
)

type taskHandler interface {
	handleTasks(cmd *msg.Cmd, args []string) ([]string, error)
	// TODO: Better options here? Write to a writer?
	describe() string
}

type noTaskHandler struct{}

func (h noTaskHandler) handleTasks(cmd *msg.Cmd, args []string) ([]string, error) {
	return args, nil
}

func (h noTaskHandler) describe() string {
	return ""
}

type singleTaskHandler struct{}

func (h singleTaskHandler) handleTasks(cmd *msg.Cmd, args []string) ([]string, error) {
	if len(args) == 0 {
		return args, errors.New("Require single task but none is given")
	}
	if tasks, err := GetTaskNames(args[0]); err != nil {
		return args, err
	} else if len(tasks) == 0 {
		return args, errors.New("Require single task but none is given")
	} else if len(tasks) > 1 {
		return args, errors.New("Require single task but several are given")
	} else if tasks[0] == AllTasks {
		return args, errors.New("Require single task name but found '" + AllTasks + "'")
	} else {
		cmd.Tasks = tasks
	}
	return args[1:], nil
}

func (h singleTaskHandler) describe() string {
	return "task"
}

type multiTaskHandler struct{}

func (h multiTaskHandler) handleTasks(cmd *msg.Cmd, args []string) ([]string, error) {
	if len(args) == 0 {
		return args, errors.New("Require one or more tasks but none is given")
	}
	if tasks, err := GetTaskNames(args[0]); err != nil {
		return args, err
	} else if len(tasks) == 0 {
		return args, errors.New("Require one or more tasks but none is given")
	} else {
		if len(tasks) > 1 {
			for _, task := range tasks {
				if task == AllTasks {
					return args, errors.New("When given, '" + AllTasks + "' must be the only task")
				}
			}
		}
		cmd.Tasks = tasks
	}
	return args[1:], nil
}

func (h multiTaskHandler) describe() string {
	return AllTasks + "|task,..."
}

type ArgHandler interface {
	// Parse the params and modify cmd accordingly.
	// Returns unused arguments and a possible error.
	HandleArgs(cmd *msg.Cmd, args []string) ([]string, error)
}

type noArgHandler struct{}

func (h noArgHandler) HandleArgs(cmd *msg.Cmd, args []string) ([]string, error) {
	return args, nil
}

// TODO: Move methods to builder?
type Parser struct {
	command     string
	taskHandler taskHandler
	argHandler  ArgHandler
}

func CommandParser(command string) *Parser {
	return &Parser{command: command, taskHandler: nil, argHandler: nil}
}

func (p *Parser) WithoutTask() *Parser {
	p.taskHandler = new(noTaskHandler)
	return p
}

func (p *Parser) WithSingleTask() *Parser {
	p.taskHandler = new(singleTaskHandler)
	return p
}

func (p *Parser) WithMultipleTasks() *Parser {
	p.taskHandler = new(multiTaskHandler)
	return p
}

func (p *Parser) WithoutParams() *Parser {
	p.argHandler = new(noArgHandler)
	return p
}

func (p *Parser) WithArgHandler(h ArgHandler) *Parser {
	p.argHandler = h
	return p
}

// Parse the given arguments.
func (p *Parser) Parse(args []string) (msg.Cmd, error) {
	cmd := msg.Cmd{Op: p.command}
	if p.taskHandler == nil {
		panic("Argument parser does not know how to handle tasks")
	}
	restArgs, err := p.taskHandler.handleTasks(&cmd, args)
	if err != nil {
		return cmd, err
	}
	if p.argHandler == nil {
		panic("Argument parser does not know how to handle parameters")
	}
	unusedArgs, err := p.argHandler.HandleArgs(&cmd, restArgs)
	if err != nil {
		return cmd, err
	} else {
		WarnUnused(unusedArgs)
		return cmd, nil
	}
}

// Warn the user about arguments being unevaluated.
// If args is empty, no warning is issued.
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
	if isParamIdentifier(name) {
		return false
	} else if hasWhitespace(name) {
		return false
	}
	return true
}

func stripKeyword(raw string) string {
	return strings.TrimLeft(raw, ":")
}

func hasWhitespace(str string) bool {
	return strings.ContainsAny(str, " \t\n")
}

// TODO: Doc comments. This one is important.
type Quantity struct {
	Type  string
	Elems []string
}

func singleQuantity(t string, elems ...string) []Quantity {
	return []Quantity{Quantity{Type: t, Elems: elems}}
}

type Param struct {
	Name        string
	RequiresArg bool
	Quantifier  Quantifier
}

type paramHandler struct {
	params map[string]Param
}

func (p paramHandler) HandleArgs(cmd *msg.Cmd, args []string) ([]string, error) {
	return args, nil
}

func HandlerForParams(params []Param) ArgHandler {
	var pmap map[string]Param
	for _, param := range params {
		if _, ok := pmap[param.Name]; ok {
			panic("Duplicate param name: " + param.Name)
		}
		pmap[param.Name] = param
	}

	return paramHandler{params: pmap}
}

func isParamIdentifier(str string) bool {
	return strings.HasPrefix(str, ":")
}

type Quantifier interface {
	Parse(str string) ([]Quantity, error)
	Describe() string
}

type ListQuantifier struct {
	elem Quantifier
}

func ListQuantifierOf(elem Quantifier) Quantifier {
	return ListQuantifier{elem}
}

func (lq ListQuantifier) Parse(str string) ([]Quantity, error) {
	qnt := []Quantity{}
	for _, part := range strings.Split(str, ",") {
		nxt, err := lq.elem.Parse(part)
		if err != nil {
			return qnt, err
		}
		qnt = append(qnt, nxt...)
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

func (pq PairQuantifier) Parse(str string) ([]Quantity, error) {
	qnt := []Quantity{}
	fields := strings.Split(str, ":")
	if len(fields) != 2 {
		return qnt, errors.New("Not a pair: " + str)
	}
	for _, part := range fields {
		nxt, err := pq.elem.Parse(part)
		if err != nil {
			return qnt, err
		}
		qnt = append(qnt, nxt...)
	}
	return qnt, nil
}

func (pq PairQuantifier) Describe() string {
	return fmt.Sprintf("%s:%[1]s", pq.elem.Describe())
}

type DateQuantifier struct{}

func (dq DateQuantifier) Parse(str string) ([]Quantity, error) {
	_, err := time.Parse("2006-01-02", str)
	return singleQuantity("day", str), err
}

func (dq DateQuantifier) Describe() string {
	return "YYYY-MM-DD"
}

type MonthQuantifier struct{}

func (mq MonthQuantifier) Parse(str string) ([]Quantity, error) {
	_, err := time.Parse("2006-01", str)
	return singleQuantity("month", str), err
}

func (mq MonthQuantifier) Describe() string {
	return "YYYY-MM"
}

type YearQuantifier struct{}

func (yq YearQuantifier) Parse(str string) ([]Quantity, error) {
	_, err := time.Parse("2006", str)
	return singleQuantity("year", str), err
}

func (yq YearQuantifier) Describe() string {
	return "YYYY"
}

type DaysAgoQuantifier struct {
	Now time.Time
}

func (daq DaysAgoQuantifier) Parse(str string) ([]Quantity, error) {
	days, err := strconv.Atoi(str)
	return singleQuantity("day", isoDate(daq.Now.AddDate(0, 0, -days))), err
}

func (daq DaysAgoQuantifier) Describe() string {
	return "N"
}

type MonthsAgoQuantifier struct {
	Now time.Time
}

func (maq MonthsAgoQuantifier) Parse(str string) ([]Quantity, error) {
	months, err := strconv.Atoi(str)
	return singleQuantity("month", isoMonth(maq.Now.AddDate(0, -months, 0))), err
}

func (maq MonthsAgoQuantifier) Describe() string {
	return "N"
}

// Format as yyyy-MM-dd.
func isoDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// Format as yyyy-MM.
func isoMonth(t time.Time) string {
	return t.Format("2006-01")
}

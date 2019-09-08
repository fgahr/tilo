// Package msg provides means for client and server to communicate.
package msg

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// Possible command arguments.
	// NOTE: They are conceivably different from request commands and therefore
	// defined separately, although identical.
	argStart    = "start"
	argStop     = "stop"
	argCurrent  = "current"
	argAbort    = "abort"
	argQuery    = "query"
	argShutdown = "shutdown"
	// Special "task" meaning show info for all tasks
	TskAllTasks = "--all"
	// Types of command for the requests
	CmdStart    = "start"
	CmdStop     = "stop"
	CmdCurrent  = "current"
	CmdAbort    = "abort"
	CmdQuery    = "query"
	CmdShutdown = "shutdown"
	// Flags and params -- no modifiers
	prmToday     = "--today"
	prmYesterday = "--yesterday"
	prmEver      = "--ever"
	prmCombine   = "--combine" // Whether to combine times for all given tasks
	// Flags and params -- modifiers required
	prmDate      = "--day"
	prmMonth     = "--month"
	prmYear      = "--year"
	prmWeeksAgo  = "--weeks-ago"
	prmMonthsAgo = "--months-ago"
	prmYearsAgo  = "--years-ago"
	prmThisWeek  = "--this-week"
	prmLastWeek  = "--last-week"
	prmThisMonth = "--this-month"
	prmLastMonth = "--last-month"
	prmThisYear  = "--this-year"
	prmLastYear  = "--last-year"
	prmSince     = "--since"
	prmBetween   = "--between"
	// Query details -- static
	QryDay   = "day"
	QryMonth = "month"
	QryYear  = "year"
	// Query details -- dynamic
	QryBetween = "between"
)

type QueryDetails []string

// Request, to be sent to the server.
// NOTE: Renaming pending as soon as the old struct is removed.
type Request struct {
	Cmd       string
	Tasks     []string
	QueryArgs []QueryDetails
	Combine   bool
}

// A request for the server to shut down.
func ShutdownRequest() Request {
	return Request{Cmd: CmdShutdown}
}

type argParser interface {
	identifier() string
	handleArgs(args []string, now time.Time) (Request, error)
}

// Create a request based on command line parameters and the current time.
// This function contains the main command language logic.
// Note that passing the time here is necessary to avoid inconsistencies when
// encountering a date change around midnight. As a side note, it also
// simplifies testing.
func ParseRequest(args []string, now time.Time) (Request, error) {
	if len(args) == 0 {
		panic("Empty argument list passed from main.")
	}
	parsers := []argParser{
		cmdOnlyParser{argStop, CmdStop, os.Stderr},
		cmdOnlyParser{argCurrent, CmdCurrent, os.Stderr},
		cmdOnlyParser{argAbort, CmdAbort, os.Stderr},
		cmdOnlyParser{argShutdown, CmdShutdown, os.Stderr},
		startParser{os.Stderr},
		queryParser{os.Stderr},
	}

	cliCmd := args[0]
	for _, p := range parsers {
		if cliCmd == p.identifier() {
			return p.handleArgs(args[1:], now)
		}
	}
	return Request{}, errors.Errorf("Unknown command: %s", args[0])
}

type cmdOnlyParser struct {
	cliCmd string
	reqCmd string
	errout io.Writer
}

func (p cmdOnlyParser) identifier() string {
	return p.cliCmd
}

func (p cmdOnlyParser) handleArgs(args []string, now time.Time) (Request, error) {
	warnIgnoredArguments(args, p.errout)
	return Request{Cmd: p.reqCmd}, nil
}

type startParser struct {
	errout io.Writer
}

func (p startParser) identifier() string {
	return argStart
}

func (p startParser) handleArgs(args []string, now time.Time) (Request, error) {
	if len(args) < 1 {
		return Request{},
			errors.New("Missing task name for 'start'.")
	}

	warnIgnoredArguments(args[1:], p.errout)

	tasks, err := getTaskNames(args[0])
	if err != nil {
		return Request{}, err
	} else if len(tasks) > 1 {
		return Request{},
			errors.Errorf("Can only start one task at a time. Given: %v", tasks)
	}
	return Request{Cmd: CmdStart, Tasks: tasks}, nil
}

type queryParser struct {
	errout io.Writer
}

func (p queryParser) identifier() string {
	return argQuery
}

// Parse args for a query request.
func (p queryParser) handleArgs(args []string, now time.Time) (Request, error) {
	if len(args) == 0 {
		return Request{},
			errors.New("Missing arguments for query request.")
	}

	tasks, err := getTaskNames(args[0])
	if err != nil {
		return Request{}, err
	}

	details, err := getQueryArgs(args[1:], now)
	if err != nil {
		return Request{}, err
	}

	request := Request{
		Cmd:       CmdQuery,
		Tasks:     tasks,
		QueryArgs: details,
		Combine:   shouldCombine(args),
	}

	return request, nil
}

// Split task names given as a comma-separated field, check for validity.
func getTaskNames(taskField string) ([]string, error) {
	if taskField == TskAllTasks {
		return []string{TskAllTasks}, nil
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

type detailParser interface {
	numberModifiers() int
	identifier() string
	parse(now time.Time, modifiers ...string) (QueryDetails, error)
}

func getDetailParsers() []detailParser {
	return []detailParser{
		noModDetailParser{id: prmToday, f: daysAgoFunc(0)},
		noModDetailParser{id: prmYesterday, f: daysAgoFunc(1)},
		noModDetailParser{id: prmThisWeek, f: weeksAgoFunc(0)},
		noModDetailParser{id: prmLastWeek, f: weeksAgoFunc(1)},
		noModDetailParser{id: prmThisMonth, f: monthsAgoFunc(0)},
		noModDetailParser{id: prmLastMonth, f: monthsAgoFunc(1)},
		noModDetailParser{id: prmThisYear, f: yearsAgoFunc(0)},
		noModDetailParser{id: prmLastYear, f: yearsAgoFunc(1)},
		singleModDetailParser{id: prmDate, f: getDate},
		singleModDetailParser{id: prmMonth, f: getMonth},
		singleModDetailParser{id: prmMonthsAgo, f: getMonthsAgo},
		singleModDetailParser{id: prmYear, f: getYear},
		singleModDetailParser{id: prmYearsAgo, f: getYearsAgo},
		singleModDetailParser{id: prmSince, f: getSince},
		betweenDetailParser{},
	}
}

// Read the extra arguments for a query request.
func getQueryArgs(args []string, now time.Time) ([]QueryDetails, error) {
	if len(args) == 0 {
		return []QueryDetails{QueryDetails{QryDay, isoDate(time.Now())}}, nil
	}

	var details []QueryDetails
	for i := 0; i < len(args); i++ {
		if args[i] == "" {
			continue
		}

		arg := strings.Split(args[i], "=")[0]
		p := findParser(arg)
		if p == nil {
			return details, errors.Errorf("No parser found for argument: %s", arg)
		}

		if p.numberModifiers() > 0 {
			modifiers := getModifiers(&i, args)
			for len(modifiers) > 0 {
				if len(modifiers) < p.numberModifiers() {
					return details, errors.Errorf("Unbalanced modifiers: %s", args[i])
				}
				d, err := p.parse(now, modifiers[0:p.numberModifiers()]...)
				if err != nil {
					return details, err
				}
				modifiers = modifiers[p.numberModifiers():]
				details = append(details, d)
			}
		} else {
			d, err := p.parse(now)
			if err != nil {
				return details, err
			}
			details = append(details, d)
		}
	}

	return details, nil
}

func findParser(arg string) detailParser {
	parsers := getDetailParsers()
	for _, p := range parsers {
		if p.identifier() == arg {
			return p
		}
	}
	return nil
}

func getModifiers(iref *int, args []string) []string {
	i := *iref
	var allMods string
	if strings.Contains(args[i], "=") {
		allMods = strings.Split(args[i], "=")[1]
	} else {
		i++
		allMods = args[i]
	}
	return strings.Split(allMods, ",")
}

type noModDetailParser struct {
	id string
	f  func(now time.Time) QueryDetails
}

func (p noModDetailParser) numberModifiers() int {
	return 0
}

func (p noModDetailParser) identifier() string {
	return p.id
}

func (p noModDetailParser) parse(now time.Time, _ ...string) (QueryDetails, error) {
	return p.f(now), nil
}

func daysAgoFunc(days int) func(time.Time) QueryDetails {
	return func(now time.Time) QueryDetails {
		return daysAgo(now, days)
	}
}

func weeksAgoFunc(weeks int) func(time.Time) QueryDetails {
	return func(now time.Time) QueryDetails {
		return weeksAgo(now, weeks)
	}
}

func monthsAgoFunc(months int) func(time.Time) QueryDetails {
	return func(now time.Time) QueryDetails {
		return monthsAgo(now, months)
	}
}

func yearsAgoFunc(years int) func(time.Time) QueryDetails {
	return func(now time.Time) QueryDetails {
		return yearsAgo(now, years)
	}
}

type singleModDetailParser struct {
	id string
	f  func(mod string, now time.Time) (QueryDetails, error)
}

func (p singleModDetailParser) numberModifiers() int {
	return 1
}

func (p singleModDetailParser) identifier() string {
	return p.id
}

func (p singleModDetailParser) parse(now time.Time, mods ...string) (QueryDetails, error) {
	if len(mods) != 1 {
		panic("Parser can only accept one modifier at a time")
	}
	return p.f(mods[0], now)
}

func getDate(mod string, _ time.Time) (QueryDetails, error) {
	if isValidIsoDate(mod) {
		return QueryDetails{QryDay, mod}, nil
	}
	return invalidDate(mod)
}

func getMonth(mod string, _ time.Time) (QueryDetails, error) {
	if isValidYearMonth(mod) {
		return QueryDetails{QryMonth, mod}, nil
	}
	return QueryDetails{}, errors.Errorf("Not a valid year-month: %s", mod)
}

func getMonthsAgo(mod string, now time.Time) (QueryDetails, error) {
	num, err := strconv.Atoi(mod)
	if err != nil {
		return QueryDetails{}, err
	}
	return monthsAgo(now, num), nil
}

func getYear(mod string, _ time.Time) (QueryDetails, error) {
	year, err := strconv.Atoi(mod)
	if err != nil {
		return QueryDetails{}, err
	}
	return QueryDetails{QryYear, fmt.Sprint(year)}, nil
}

func getYearsAgo(mod string, now time.Time) (QueryDetails, error) {
	num, err := strconv.Atoi(mod)
	if err != nil {
		return QueryDetails{}, err
	}
	return yearsAgo(now, num), nil
}

func getSince(mod string, now time.Time) (QueryDetails, error) {
	if isValidIsoDate(mod) {
		return QueryDetails{QryBetween, mod, isoDate(now)}, nil
	}
	return invalidDate(mod)
}

type betweenDetailParser struct{}

func (p betweenDetailParser) identifier() string {
	return prmBetween
}

func (p betweenDetailParser) numberModifiers() int {
	return 2
}

func (p betweenDetailParser) parse(now time.Time, mods ...string) (QueryDetails, error) {
	if len(mods) != 2 {
		panic("Parser must be given two modifiers at a time")
	}
	d1 := mods[0]
	d2 := mods[1]
	if !isValidIsoDate(d1) {
		return invalidDate(d1)
	}
	if !isValidIsoDate(d2) {
		return invalidDate(d2)
	}
	return QueryDetails{QryBetween, d1, d2}, nil
}

func invalidDate(s string) (QueryDetails, error) {
	return QueryDetails{}, errors.Errorf("Not a valid date: %s", s)
}

// Whether to combine results for all tasks
func shouldCombine(args []string) bool {
	for i, arg := range args {
		if arg == prmCombine {
			args[i] = ""
			return true
		}
	}
	return false
}

// If args are given, a warning is emitted that they will be ignored.
func warnIgnoredArguments(args []string, out io.Writer) {
	if len(args) > 0 {
		fmt.Fprintf(out, "Extra arguments ignored: %v", args)
	}
}

// Detail describing a a date a number of days ago.
func daysAgo(now time.Time, days int) QueryDetails {
	day := now.AddDate(0, 0, -days).Format("2006-01-02")
	return QueryDetails{QryDay, day}
}

// Detail describing the week (Mon-Sun) the given number of weeks ago.
func weeksAgo(now time.Time, weeks int) QueryDetails {
	daysSinceLastMonday := (int(now.Weekday()) + 6) % 7
	// Monday in the target week
	start := now.AddDate(0, 0, -(daysSinceLastMonday + 7*weeks))
	// Sunday
	end := start.AddDate(0, 0, 6)
	// Avoid passing a future date.
	if end.After(now) {
		end = now
	}
	return QueryDetails{QryBetween, isoDate(start), isoDate(end)}
}

// Detail describing the month (1st to last) the given number of months ago.
func monthsAgo(now time.Time, months int) QueryDetails {
	// NOTE: Simply going back the given amount of months could result in
	// "overflowing" to the next month, e.g. May 31st going back 1 month
	// is April 31st, in turn becoming May 1st. Hence normalize to the first.
	firstInMonth := now.AddDate(0, -months, -(now.Day() - 1))
	return QueryDetails{QryMonth, firstInMonth.Format("2006-01")}
}

// Detail describing the full year the given number of years ago.
func yearsAgo(now time.Time, years int) QueryDetails {
	start := now.AddDate(-years, 0, 0)
	return QueryDetails{QryYear, start.Format("2006")}
}

// Format as yyyy-MM-dd.
func isoDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// Parse a comma-separated list of dates as query details.
func getDays(s string) ([]QueryDetails, bool) {
	dates, ok := getDates(s)
	if !ok {
		return nil, false
	}
	var details []QueryDetails
	for _, date := range dates {
		details = append(details, QueryDetails{QryDay, date})
	}
	return details, true
}

// Extract date strings from a comma-separated list.
func getDates(s string) ([]string, bool) {
	rawDates := strings.Split(s, ",")
	var dates []string
	for _, date := range rawDates {
		if !isValidIsoDate(date) {
			return nil, false
		}
		dates = append(dates, date)
	}
	return dates, true
}

// Whether the string describes an ISO formatted date yyyy-MM-dd.
func isValidIsoDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// Whether the string describes a year and month as yyyy-MM
func isValidYearMonth(s string) bool {
	_, err := time.Parse("2006-01", s)
	return err == nil
}

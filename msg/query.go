// Package msg provides means for client and server to communicate.
package msg

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	// Special "task" meaning show info for all tasks
	TskAllTasks = "--all"
	// Flags and params -- no modifiers
	PrmToday     = "--today"
	PrmYesterday = "--yesterday"
	PrmEver      = "--ever"
	PrmCombine   = "--combine" // Whether to combine times for all given tasks
	// Flags and params -- modifiers required
	PrmDate      = "--day"
	PrmMonth     = "--month"
	PrmYear      = "--year"
	PrmWeeksAgo  = "--weeks-ago"
	PrmMonthsAgo = "--months-ago"
	PrmYearsAgo  = "--years-ago"
	PrmThisWeek  = "--this-week"
	PrmLastWeek  = "--last-week"
	PrmThisMonth = "--this-month"
	PrmLastMonth = "--last-month"
	PrmThisYear  = "--this-year"
	PrmLastYear  = "--last-year"
	PrmSince     = "--since"
	PrmBetween   = "--between"
	// Query details -- static
	QryDay   = "day"
	QryMonth = "month"
	QryYear  = "year"
	// Query details -- dynamic
	QryBetween = "between"
)

type queryParser struct {
	errout io.Writer
}

func (p queryParser) identifier() string {
	return ArgQuery
}

// Parse args for a query request.
func (p queryParser) handleArgs(args []string, now time.Time) (string, Request, error) {
	if len(args) == 0 {
		return "", Request{},
			errors.New("Missing arguments for query request.")
	}

	tasks, err := getTaskNames(args[0])
	if err != nil {
		return "", Request{}, errors.Wrap(err, "Unable to determine task names")
	}

	details, err := getQueryArgs(args[1:], now)
	if err != nil {
		return "", Request{}, errors.Wrap(err, "Unable to determine query arguments")
	}

	request := Request{
		Cmd:       CmdQuery,
		Tasks:     tasks,
		QueryArgs: details,
		Combine:   shouldCombine(args),
	}

	return "RequestHandler.Query", request, nil
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
		noModDetailParser{id: PrmToday, f: daysAgoFunc(0)},
		noModDetailParser{id: PrmYesterday, f: daysAgoFunc(1)},
		noModDetailParser{id: PrmThisWeek, f: weeksAgoFunc(0)},
		noModDetailParser{id: PrmLastWeek, f: weeksAgoFunc(1)},
		noModDetailParser{id: PrmThisMonth, f: monthsAgoFunc(0)},
		noModDetailParser{id: PrmLastMonth, f: monthsAgoFunc(1)},
		noModDetailParser{id: PrmThisYear, f: yearsAgoFunc(0)},
		noModDetailParser{id: PrmLastYear, f: yearsAgoFunc(1)},
		noModDetailParser{id: PrmEver, f: getSinceEpoch},
		singleModDetailParser{id: PrmDate, f: getDate},
		singleModDetailParser{id: PrmMonth, f: getMonth},
		singleModDetailParser{id: PrmMonthsAgo, f: getMonthsAgo},
		singleModDetailParser{id: PrmYear, f: getYear},
		singleModDetailParser{id: PrmYearsAgo, f: getYearsAgo},
		singleModDetailParser{id: PrmSince, f: getSince},
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

func getSinceEpoch(now time.Time) QueryDetails {
	details, _ := getSince("1970-01-01", now)
	return details
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
	return PrmBetween
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
		if arg == PrmCombine {
			args[i] = ""
			return true
		}
	}
	return false
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

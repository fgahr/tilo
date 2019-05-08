// Package msg provides means for client and server to communicate.
package msg

import (
	"fmt"
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

// Create a request based on command line parameters and the current time.
// This function contains the main command language logic.
// Note that passing the time here is necessary to avoid inconsistencies when
// encountering a date change around midnight. As a side note, it also
// simplifies testing.
func ParseRequest(args []string, now time.Time) (Request, error) {
	if len(args) == 0 {
		panic("Empty argument list passed from main.")
	}

	switch args[0] {
	case argStart:
		return parseStart(args[1:])
	case argStop:
		return parseCmdOnly(CmdStop, args[1:])
	case argCurrent:
		return parseCmdOnly(CmdCurrent, args[1:])
	case argAbort:
		return parseCmdOnly(CmdAbort, args[1:])
	case argQuery:
		return parseQuery(args[1:], now)
	case argShutdown:
		return parseCmdOnly(CmdShutdown, args[1:])
	default:
		return Request{}, fmt.Errorf("Unknown command: %s", args[0])
	}
}

// Parse args for a start request.
func parseStart(args []string) (Request, error) {
	if len(args) < 1 {
		return Request{},
			fmt.Errorf("Missing task name for 'start'.")
	}

	warnIgnoredArguments(args[1:])

	tasks, err := getTaskNames(args[0])
	if err != nil {
		return Request{}, err
	} else if len(tasks) > 1 {
		return Request{},
			fmt.Errorf("Can only start one task at a time. Given: %v", tasks)
	}
	return Request{Cmd: CmdStart, Tasks: tasks}, nil
}

// Parse a request containing only a command. Extra arguments will be ignored.
func parseCmdOnly(cmd string, args []string) (Request, error) {
	warnIgnoredArguments(args)
	return Request{Cmd: cmd}, nil
}

// Parse args for a query request.
func parseQuery(args []string, now time.Time) (Request, error) {
	if len(args) == 0 {
		return Request{},
			fmt.Errorf("Missing arguments for query request.")
	}

	request := Request{Cmd: CmdQuery}

	tasks, err := getTaskNames(args[0])
	if err != nil {
		return request, err
	}

	details, err := getQueryArgs(args[1:], now)
	if err != nil {
		return request, err
	}

	request.Tasks = tasks
	request.QueryArgs = details
	request.Combine = shouldCombine(args)

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
			return nil, fmt.Errorf("Invalid task name: %s", task)
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

// Read the extra arguments for a query request.
func getQueryArgs(args []string, now time.Time) ([]QueryDetails, error) {
	if len(args) == 0 {
		return []QueryDetails{QueryDetails{QryDay, isoDate(time.Now())}}, nil
	}
	var details []QueryDetails
	for _, arg := range args {
		if strings.Contains(arg, "=") {
			parsed, err := parseWithModifier(arg, now)
			if err != nil {
				return nil, err
			}
			for _, p := range parsed {
				details = append(details, p)
			}
		} else {
			parsed, err := parseWithoutModifiers(arg, now)
			if err != nil {
				return nil, err
			}
			details = append(details, parsed)
		}
	}
	return details, nil
}

// Translate arguments which contain no modifiers, such as `--today`.
func parseWithoutModifiers(arg string, now time.Time) (QueryDetails, error) {
	switch arg {
	case prmToday:
		return daysAgo(now, 0), nil
	case prmYesterday:
		return daysAgo(now, 1), nil
	case prmThisWeek:
		return weeksAgo(now, 0), nil
	case prmLastWeek:
		return weeksAgo(now, 1), nil
	case prmThisMonth:
		return monthsAgo(now, 0), nil
	case prmLastMonth:
		return monthsAgo(now, 1), nil
	case prmThisYear:
		return yearsAgo(now, 0), nil
	case prmLastYear:
		return yearsAgo(now, 1), nil
	default:
		return nil, fmt.Errorf("Unknown flag: %s", arg)
	}
}

// Parse an argument of the form `--detail=modifier`
func parseWithModifier(arg string, now time.Time) ([]QueryDetails, error) {
	detailAndModifier := strings.Split(arg, "=")
	detail := detailAndModifier[0]
	modifier := detailAndModifier[1]

	switch detail {
	case prmDate:
		if dates, ok := getDays(modifier); ok {
			return dates, nil
		}
		return nil, fmt.Errorf("Must be a date or list of dates: %s", modifier)
	case prmMonth:
		if months, ok := getMonths(modifier); ok {
			return months, nil
		}
		return nil, fmt.Errorf("Must be a month or list of months: %s", modifier)
	case prmMonthsAgo:
		if nums, ok := getNumbers(modifier); ok {
			var details []QueryDetails
			for _, n := range nums {
				details = append(details, monthsAgo(now, n))
			}
			return details, nil
		}
		return nil, fmt.Errorf("Must be a number or list of numbers: %s", modifier)
	case prmYear:
		if nums, ok := getNumbers(modifier); ok {
			var details []QueryDetails
			for _, year := range nums {
				details = append(details, QueryDetails{QryYear, fmt.Sprint(year)})
			}
			return details, nil
		}
		return nil, fmt.Errorf("Must be year or list of years: %s", modifier)
	case prmYearsAgo:
		if nums, ok := getNumbers(modifier); ok {
			var details []QueryDetails
			currentYear := now.Year()
			for _, n := range nums {
				details = append(details, QueryDetails{QryYear,
					fmt.Sprint(currentYear - n)})
			}
			return details, nil
		}
		return nil, fmt.Errorf("Must be a number or list of numbers: %s", modifier)
	case prmSince:
		today := isoDate(now)
		if startDates, ok := getDates(modifier); ok {
			var details []QueryDetails
			for _, start := range startDates {
				details = append(details, QueryDetails{QryBetween, start, today})
			}
			return details, nil
		}
		return nil, fmt.Errorf("Must be a date or list of dates: %s", modifier)
	case prmBetween:
		if dates, ok := getDates(modifier); ok {
			if len(dates) != 2 {
				return nil, fmt.Errorf("Must be a list of two dates: %s", modifier)
			}
			return []QueryDetails{
				QueryDetails{QryBetween, dates[0], dates[1]}}, nil
		}
		return nil, fmt.Errorf("Must be a list of two dates: %s", modifier)
	default:
		return nil, fmt.Errorf("Unknown query detail: %s", detail)
	}
}

// Whether to combine results for all tasks
func shouldCombine(args []string) bool {
	for _, arg := range args {
		if arg == prmCombine {
			return true
		}
	}
	return false
}

// If args are given, a warning is emitted that they will be ignored.
func warnIgnoredArguments(args []string) {
	if len(args) > 0 {
		fmt.Fprintf(os.Stderr, "Extra arguments ignored: %v", args)
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

// Takes a list of months as yyyy-MM and returns the first for each.
func getMonths(s string) ([]QueryDetails, bool) {
	months := strings.Split(s, ",")
	var details []QueryDetails
	for _, month := range months {
		_, err := time.Parse("2006-01", month)
		if err != nil {
			return nil, false
		}
		details = append(details, QueryDetails{QryMonth, month})
	}
	return details, true
}

// Parse a comma-separated list of numbers.
func getNumbers(s string) ([]int, bool) {
	var numbers []int
	for _, num := range strings.Split(s, ",") {
		n, err := strconv.Atoi(num)
		if err != nil {
			return nil, false
		}
		numbers = append(numbers, n)
	}
	return numbers, true
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

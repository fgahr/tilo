package query

import (
	"fmt"
	arg "github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/argparse/quantifier"
	"github.com/fgahr/tilo/msg"
	"github.com/pkg/errors"
	"strconv"
	"strings"
	"time"
)

const (
	// Special "task" meaning show info for all tasks
	TskAllTasks = arg.ParamIdentifierPrefix + "all"
	// Flags and params -- no modifiers
	paramToday     = "today"
	paramYesterday = "yesterday"
	paramEver      = "ever"
	// Flags and params -- modifiers required
	paramDay       = "day"
	paramMonth     = "month"
	paramYear      = "year"
	paramDaysAgo   = "days-ago"
	paramWeeksAgo  = "weeks-ago"
	paramMonthsAgo = "months-ago"
	paramYearsAgo  = "years-ago"
	paramThisWeek  = "this-week"
	paramLastWeek  = "last-week"
	paramThisMonth = "this-month"
	paramLastMonth = "last-month"
	paramThisYear  = "this-year"
	paramLastYear  = "last-year"
	paramSince     = "since"
	paramBetween   = "between"
	// Query details -- static
	// QryDay   = "day"
	// QryMonth = "month"
	// QryYear  = "year"
	// // Query details -- dynamic
	// QryBetween = "between"
	// PrmCombine   = ":combine" // Whether to combine times for all given tasks
)

type queryArgHandler struct {
	now    time.Time
	params map[string]arg.Param
}

func (h *queryArgHandler) registerParam(param arg.Param) {
	if _, ok := h.params[param.Name]; ok {
		panic("Duplicate parameter name: " + param.Name)
	}
	h.params[param.Name] = param
}

func (h *queryArgHandler) HandleArgs(cmd *msg.Cmd, params []string) ([]string, error) {
	parseQueryArgs(params, cmd)
	return nil, nil
}

func newQueryArgHandler(now time.Time) *queryArgHandler {
	h := &queryArgHandler{now: now}
	params := []arg.Param{
		// Fixed day
		arg.Param{
			Name:        paramToday,
			RequiresArg: false,
			Quantifier:  quantifier.FixedDayOffset(now, 0),
			Description: "Today's activity",
		},
		arg.Param{
			Name:        paramYesterday,
			RequiresArg: false,
			Quantifier:  quantifier.FixedDayOffset(now, -1),
			Description: "Yesterday's activity",
		},

		// Fixed week
		arg.Param{
			Name:        paramThisWeek,
			RequiresArg: false,
			Quantifier:  quantifier.FixedWeekOffset(now, 0),
			Description: "This week's activity",
		},
		arg.Param{
			Name:        paramLastWeek,
			RequiresArg: false,
			Quantifier:  quantifier.FixedWeekOffset(now, -1),
			Description: "Last week's activity",
		},

		// Fixed month
		arg.Param{
			Name:        paramThisMonth,
			RequiresArg: false,
			Quantifier:  quantifier.FixedMonthOffset(now, 0),
			Description: "This month's activity",
		},
		arg.Param{
			Name:        paramLastMonth,
			RequiresArg: false,
			Quantifier:  quantifier.FixedMonthOffset(now, -1),
			Description: "Last month's activity",
		},

		// Fixed year
		arg.Param{
			Name:        paramThisYear,
			RequiresArg: false,
			Quantifier:  quantifier.FixedYearOffset(now, 0),
			Description: "This year's activity",
		},
		arg.Param{
			Name:        paramLastYear,
			RequiresArg: false,
			Quantifier:  quantifier.FixedYearOffset(now, -1),
			Description: "Last year's activity",
		},

		// Dynamic day/week/month/year
		arg.Param{
			Name:        paramDaysAgo,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.DynamicDayOffset(now)),
			Description: "Activity N days ago.",
		},
		arg.Param{
			Name:        paramWeeksAgo,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.DynamicWeekOffset(now)),
			Description: "Activity N weeks ago.",
		},
		arg.Param{
			Name:        paramMonthsAgo,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.DynamicMonthOffset(now)),
			Description: "Activity N months ago.",
		},
		arg.Param{
			Name:        paramYearsAgo,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.DynamicYearOffset(now)),
			Description: "Activity N years ago.",
		},

		// Specific day/month/year
		arg.Param{
			Name:        paramDay,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.SpecificDate()),
			Description: "Activity on a given day",
		},
		arg.Param{
			Name:        paramMonth,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.SpecificMonth()),
			Description: "Activity in a given month",
		},
		arg.Param{
			Name:        paramYear,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.SpecificYear()),
			Description: "Activity in a given year",
		},

		// Interval since/between
		// TODO
	}

	for _, param := range params {
		h.registerParam(param)
	}

	return h
}

func parseQueryArgs(args []string, cmd *msg.Cmd) error {
	now := time.Now()
	if len(args) == 0 {
		return errors.New("Missing arguments for query request.")
	}

	if params, err := getQueryParams(args, now); err != nil {
		return errors.Wrap(err, "Unable to parse query arguments")
	} else {
		cmd.Quantities = params
	}
	return nil
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
	parse(now time.Time, modifiers ...string) (msg.QueryParam, error)
}

func getDetailParsers() []detailParser {
	return []detailParser{
		noModDetailParser{id: paramToday, f: daysAgoFunc(0)},
		noModDetailParser{id: paramYesterday, f: daysAgoFunc(1)},
		noModDetailParser{id: paramThisWeek, f: weeksAgoFunc(0)},
		noModDetailParser{id: paramLastWeek, f: weeksAgoFunc(1)},
		noModDetailParser{id: paramThisMonth, f: monthsAgoFunc(0)},
		noModDetailParser{id: paramLastMonth, f: monthsAgoFunc(1)},
		noModDetailParser{id: paramThisYear, f: yearsAgoFunc(0)},
		noModDetailParser{id: paramLastYear, f: yearsAgoFunc(1)},
		noModDetailParser{id: paramEver, f: getSinceEpoch},
		singleModDetailParser{id: paramDay, f: getDate},
		singleModDetailParser{id: paramMonth, f: getMonth},
		singleModDetailParser{id: paramMonthsAgo, f: getMonthsAgo},
		singleModDetailParser{id: paramYear, f: getYear},
		singleModDetailParser{id: paramYearsAgo, f: getYearsAgo},
		singleModDetailParser{id: paramSince, f: getSince},
		betweenDetailParser{},
	}
}

// Read the extra arguments for a query request.
func getQueryParams(args []string, now time.Time) ([]msg.Quantity, error) {
	panic("Calling obsolete method getQueryParams")

	// var details []msg.QueryParam
	// for i := 0; i < len(args); i++ {
	//	if args[i] == "" {
	//		continue
	//	}

	//	arg := strings.Split(args[i], "=")[0]
	//	p := findParser(arg)
	//	if p == nil {
	//		return details, errors.Errorf("No parser found for argument: %s", arg)
	//	}

	//	if p.numberModifiers() > 0 {
	//		modifiers := getModifiers(&i, args)
	//		for len(modifiers) > 0 {
	//			if len(modifiers) < p.numberModifiers() {
	//				return details, errors.Errorf("Unbalanced modifiers: %s", args[i])
	//			}
	//			d, err := p.parse(now, modifiers[0:p.numberModifiers()]...)
	//			if err != nil {
	//				return details, err
	//			}
	//			modifiers = modifiers[p.numberModifiers():]
	//			details = append(details, d)
	//		}
	//	} else {
	//		d, err := p.parse(now)
	//		if err != nil {
	//			return details, err
	//		}
	//		details = append(details, d)
	//	}
	// }

	// return details, nil
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
	f  func(now time.Time) msg.QueryParam
}

func (p noModDetailParser) numberModifiers() int {
	return 0
}

func (p noModDetailParser) identifier() string {
	return p.id
}

func (p noModDetailParser) parse(now time.Time, _ ...string) (msg.QueryParam, error) {
	return p.f(now), nil
}

func daysAgoFunc(days int) func(time.Time) msg.QueryParam {
	return func(now time.Time) msg.QueryParam {
		return daysAgo(now, days)
	}
}

func weeksAgoFunc(weeks int) func(time.Time) msg.QueryParam {
	return func(now time.Time) msg.QueryParam {
		return weeksAgo(now, weeks)
	}
}

func monthsAgoFunc(months int) func(time.Time) msg.QueryParam {
	return func(now time.Time) msg.QueryParam {
		return monthsAgo(now, months)
	}
}

func yearsAgoFunc(years int) func(time.Time) msg.QueryParam {
	return func(now time.Time) msg.QueryParam {
		return yearsAgo(now, years)
	}
}

func getSinceEpoch(now time.Time) msg.QueryParam {
	details, _ := getSince("1970-01-01", now)
	return details
}

type singleModDetailParser struct {
	id string
	f  func(mod string, now time.Time) (msg.QueryParam, error)
}

func (p singleModDetailParser) numberModifiers() int {
	return 1
}

func (p singleModDetailParser) identifier() string {
	return p.id
}

func (p singleModDetailParser) parse(now time.Time, mods ...string) (msg.QueryParam, error) {
	if len(mods) != 1 {
		panic("Parser can only accept one modifier at a time")
	}
	return p.f(mods[0], now)
}

func getDate(mod string, _ time.Time) (msg.QueryParam, error) {
	if isValidIsoDate(mod) {
		return msg.QueryParam{quantifier.TimeDay, mod}, nil
	}
	return invalidDate(mod)
}

func getMonth(mod string, _ time.Time) (msg.QueryParam, error) {
	if isValidYearMonth(mod) {
		return msg.QueryParam{quantifier.TimeMonth, mod}, nil
	}
	return msg.QueryParam{}, errors.Errorf("Not a valid year-month: %s", mod)
}

func getMonthsAgo(mod string, now time.Time) (msg.QueryParam, error) {
	num, err := strconv.Atoi(mod)
	if err != nil {
		return msg.QueryParam{}, err
	}
	return monthsAgo(now, num), nil
}

func getYear(mod string, _ time.Time) (msg.QueryParam, error) {
	year, err := strconv.Atoi(mod)
	if err != nil {
		return msg.QueryParam{}, err
	}
	return msg.QueryParam{quantifier.TimeYear, fmt.Sprint(year)}, nil
}

func getYearsAgo(mod string, now time.Time) (msg.QueryParam, error) {
	num, err := strconv.Atoi(mod)
	if err != nil {
		return msg.QueryParam{}, err
	}
	return yearsAgo(now, num), nil
}

func getSince(mod string, now time.Time) (msg.QueryParam, error) {
	if isValidIsoDate(mod) {
		return msg.QueryParam{quantifier.TimeBetween, mod, isoDate(now)}, nil
	}
	return invalidDate(mod)
}

type betweenDetailParser struct{}

func (p betweenDetailParser) identifier() string {
	return paramBetween
}

func (p betweenDetailParser) numberModifiers() int {
	return 2
}

func (p betweenDetailParser) parse(now time.Time, mods ...string) (msg.QueryParam, error) {
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
	return msg.QueryParam{quantifier.TimeBetween, d1, d2}, nil
}

func invalidDate(s string) (msg.QueryParam, error) {
	return msg.QueryParam{}, errors.Errorf("Not a valid date: %s", s)
}

// Whether to combine results for all tasks
func shouldCombine(args []string) bool {
	// NOTE: Currently disabled.
	// for i, arg := range args {
	//	if arg == PrmCombine {
	//		args[i] = ""
	//		return true
	//	}
	// }
	return false
}

// Detail describing a a date a number of days ago.
func daysAgo(now time.Time, days int) msg.QueryParam {
	day := now.AddDate(0, 0, -days).Format("2006-01-02")
	return msg.QueryParam{quantifier.TimeDay, day}
}

// Detail describing the week (Mon-Sun) the given number of weeks ago.
func weeksAgo(now time.Time, weeks int) msg.QueryParam {
	daysSinceLastMonday := (int(now.Weekday()) + 6) % 7
	// Monday in the target week
	start := now.AddDate(0, 0, -(daysSinceLastMonday + 7*weeks))
	// Sunday
	end := start.AddDate(0, 0, 6)
	// Avoid passing a future date.
	if end.After(now) {
		end = now
	}
	return msg.QueryParam{quantifier.TimeBetween, isoDate(start), isoDate(end)}
}

// Detail describing the month (1st to last) the given number of months ago.
func monthsAgo(now time.Time, months int) msg.QueryParam {
	// NOTE: Simply going back the given amount of months could result in
	// "overflowing" to the next month, e.g. May 31st going back 1 month
	// is April 31st, in turn becoming May 1st. Hence normalize to the first.
	firstInMonth := now.AddDate(0, -months, -(now.Day() - 1))
	return msg.QueryParam{quantifier.TimeMonth, firstInMonth.Format("2006-01")}
}

// Detail describing the full year the given number of years ago.
func yearsAgo(now time.Time, years int) msg.QueryParam {
	start := now.AddDate(-years, 0, 0)
	return msg.QueryParam{quantifier.TimeYear, start.Format("2006")}
}

// Parse a comma-separated list of dates as query details.
func getDays(s string) ([]msg.QueryParam, bool) {
	dates, ok := getDates(s)
	if !ok {
		return nil, false
	}
	var details []msg.QueryParam
	for _, date := range dates {
		details = append(details, msg.QueryParam{quantifier.TimeDay, date})
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

// Format as yyyy-MM-dd.
func isoDate(t time.Time) string {
	return t.Format("2006-01-02")
}

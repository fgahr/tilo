package query

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/argparse/quantifier"
	"time"
)

const (
	// Special "task" meaning show info for all tasks
	TskAllTasks = argparse.ParamIdentifierPrefix + "all"
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
)

func newQueryArgHandler(now time.Time) argparse.ArgHandler {
	params := []argparse.Param{
		// Fixed day
		argparse.Param{
			Name:        paramToday,
			RequiresArg: false,
			Quantifier:  quantifier.FixedDayOffset(now, 0),
			Description: "Today's activity",
		},
		argparse.Param{
			Name:        paramYesterday,
			RequiresArg: false,
			Quantifier:  quantifier.FixedDayOffset(now, -1),
			Description: "Yesterday's activity",
		},

		// Fixed week
		argparse.Param{
			Name:        paramThisWeek,
			RequiresArg: false,
			Quantifier:  quantifier.FixedWeekOffset(now, 0),
			Description: "This week's activity",
		},
		argparse.Param{
			Name:        paramLastWeek,
			RequiresArg: false,
			Quantifier:  quantifier.FixedWeekOffset(now, -1),
			Description: "Last week's activity",
		},

		// Fixed month
		argparse.Param{
			Name:        paramThisMonth,
			RequiresArg: false,
			Quantifier:  quantifier.FixedMonthOffset(now, 0),
			Description: "This month's activity",
		},
		argparse.Param{
			Name:        paramLastMonth,
			RequiresArg: false,
			Quantifier:  quantifier.FixedMonthOffset(now, -1),
			Description: "Last month's activity",
		},

		// Fixed year
		argparse.Param{
			Name:        paramThisYear,
			RequiresArg: false,
			Quantifier:  quantifier.FixedYearOffset(now, 0),
			Description: "This year's activity",
		},
		argparse.Param{
			Name:        paramLastYear,
			RequiresArg: false,
			Quantifier:  quantifier.FixedYearOffset(now, -1),
			Description: "Last year's activity",
		},

		// Dynamic day/week/month/year
		argparse.Param{
			Name:        paramDaysAgo,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.DynamicDayOffset(now)),
			Description: "Activity N days ago.",
		},
		argparse.Param{
			Name:        paramWeeksAgo,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.DynamicWeekOffset(now)),
			Description: "Activity N weeks ago.",
		},
		argparse.Param{
			Name:        paramMonthsAgo,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.DynamicMonthOffset(now)),
			Description: "Activity N months ago.",
		},
		argparse.Param{
			Name:        paramYearsAgo,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.DynamicYearOffset(now)),
			Description: "Activity N years ago.",
		},

		// Specific day/month/year
		argparse.Param{
			Name:        paramDay,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.SpecificDate()),
			Description: "Activity on a given day",
		},
		argparse.Param{
			Name:        paramMonth,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.SpecificMonth()),
			Description: "Activity in a given month",
		},
		argparse.Param{
			Name:        paramYear,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.SpecificYear()),
			Description: "Activity in a given year",
		},

		// Interval since/between
		argparse.Param{
			Name:        paramSince,
			RequiresArg: true,
			Quantifier:  quantifier.ListOf(quantifier.DynamicUntil(now)),
			Description: "Activity since a specific day",
		},
		argparse.Param{
			Name:        paramBetween,
			RequiresArg: true,
			Quantifier: quantifier.ListOf(
				quantifier.TaggedPair(quantifier.TimeBetween, quantifier.SpecificDate())),
			Description: "Activity between two dates",
		},
	}

	return argparse.HandlerForParams(params)
}

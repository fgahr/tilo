// Package msg provides means for client and server to communicate.
package msg

import (
	"reflect"
	"testing"
	"time"
)

// Reference time for testing.
var now time.Time = time.Date(2019, 1, 8, 12, 0, 0, 0, time.UTC)

const (
	today     = "2019-01-08"
	yesterday = "2019-01-07"
)

func singleDetail(details ...string) []QueryDetails {
	return []QueryDetails{details}
}

func parseAndCheckExpected(t *testing.T, want Request, args ...string) {
	have, err := ParseRequest(args, now)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(want, have) {
		t.Errorf("Wanted %v but got: %v", want, have)
	}
}

func parseShouldFail(t *testing.T, args ...string) {
	_, err := ParseRequest(args, now)
	if err == nil {
		t.Error("Operation expected to fail but succeeded with args:", args)
	}
}

func TestStartRequest(t *testing.T) {
	var want Request
	// start foo
	want = Request{Cmd: CmdStart, Tasks: []string{"foo"}}
	parseAndCheckExpected(t, want, argStart, "foo")
	// start bar
	want = Request{Cmd: CmdStart, Tasks: []string{"bar"}}
	parseAndCheckExpected(t, want, argStart, "bar")
	// start foo-bar
	want = Request{Cmd: CmdStart, Tasks: []string{"foo-bar"}}
	parseAndCheckExpected(t, want, argStart, "foo-bar")

	// start foo,bar
	parseShouldFail(t, argStart, "foo,bar")
	parseShouldFail(t, argStart, "--cool-task")
	parseShouldFail(t, argStart, "cool task")
}

func TestStopRequest(t *testing.T) {
	var want Request
	// stop
	want = Request{Cmd: CmdStop}
	parseAndCheckExpected(t, want, argStop)
}

func TestCurrentRequest(t *testing.T) {
	want := Request{Cmd: CmdCurrent}
	parseAndCheckExpected(t, want, argCurrent)
}

func TestDiscardRequest(t *testing.T) {
	want := Request{Cmd: CmdAbort}
	parseAndCheckExpected(t, want, argAbort)
}

func TestQueryRequestWithoutModifiers(t *testing.T) {
	var want Request
	// Single task, single day
	want = Request{
		Cmd:       CmdQuery,
		Tasks:     []string{"foo"},
		QueryArgs: []QueryDetails{QueryDetails{"day", today}},
		Combine:   false,
	}
	parseAndCheckExpected(t, want, argQuery, "foo", prmToday)
	// Query all tasks
	want = Request{
		Cmd:       CmdQuery,
		Tasks:     []string{TskAllTasks},
		QueryArgs: []QueryDetails{QueryDetails{"day", today}},
		Combine:   false,
	}
	parseAndCheckExpected(t, want, argQuery, TskAllTasks, prmToday)
	// Several tasks, single day
	want = Request{
		Cmd:       CmdQuery,
		Tasks:     []string{"foo", "bar", "baz"},
		QueryArgs: singleDetail(QryDay, yesterday),
		Combine:   false,
	}
	parseAndCheckExpected(t, want, argQuery, "foo,bar,baz", prmYesterday)
	// Single task, several days
	task := "a-simple-task"
	want = Request{
		Cmd:   CmdQuery,
		Tasks: []string{task},
		QueryArgs: []QueryDetails{
			QueryDetails{QryDay, today},
			QueryDetails{QryDay, yesterday},
		},
		Combine: false,
	}
	parseAndCheckExpected(t, want, argQuery, task, prmToday, prmYesterday)
	// --this-month
	want.QueryArgs = singleDetail(QryMonth, "2019-01")
	parseAndCheckExpected(t, want, argQuery, task, prmThisMonth)
	// --last-month
	want.QueryArgs = singleDetail(QryMonth, "2018-12")
	parseAndCheckExpected(t, want, argQuery, task, prmLastMonth)
	// --last-month --today
	want.QueryArgs = []QueryDetails{
		QueryDetails{QryMonth, "2018-12"},
		QueryDetails{QryDay, today},
	}
	parseAndCheckExpected(t, want, argQuery, task, prmLastMonth, prmToday)
	// --this-week. Note that `today` (2019-01-08) is a Tuesday
	want.QueryArgs = singleDetail(QryBetween, "2019-01-07", today)
	parseAndCheckExpected(t, want, argQuery, task, prmThisWeek)
	// --last-week. Note that `today` (2019-01-08) is a Tuesday
	want.QueryArgs = singleDetail(QryBetween, "2018-12-31", "2019-01-06")
	parseAndCheckExpected(t, want, argQuery, task, prmLastWeek)
	// --this-year
	want.QueryArgs = singleDetail(QryYear, "2019")
	parseAndCheckExpected(t, want, argQuery, task, prmThisYear)
	// --last-year
	want.QueryArgs = singleDetail(QryYear, "2018")
	parseAndCheckExpected(t, want, argQuery, task, prmLastYear)

	// Invalid tasks
	parseShouldFail(t, argQuery, "foo,--bar")
	parseShouldFail(t, argQuery, "foo , bar")
	parseShouldFail(t, argQuery, "--task")
	// Invalid details
	parseShouldFail(t, argQuery, "realtask", "--not-a-real-param")
}

func TestQueryRequestFailInvalidModifiers(t *testing.T) {
	task := "task"
	parseShouldFail(t, argQuery, task, "--day=2019-02-31")
	parseShouldFail(t, argQuery, task, "--day=2019--2-31")
	parseShouldFail(t, argQuery, task, "--day=2019-12-1")
	parseShouldFail(t, argQuery, task, "--day=foo")
	parseShouldFail(t, argQuery, task, "--month=2018-14")
	parseShouldFail(t, argQuery, task, "--month=2013")
	parseShouldFail(t, argQuery, task, "--month=foo")
	parseShouldFail(t, argQuery, task, "--year=last")
	parseShouldFail(t, argQuery, task, "--year=2017-02")
	parseShouldFail(t, argQuery, task, "--year=foo")
	parseShouldFail(t, argQuery, task, "--between=2017-01-01")
	parseShouldFail(t, argQuery, task, "--between=2017-01-01,2018-12-31,2019-04-26")
	parseShouldFail(t, argQuery, task, "--between=2017-01-01,foo")
	parseShouldFail(t, argQuery, task, "--between=foo")
}

func TestQueryRequestDay(t *testing.T) {
	var want Request
	// Single task and day
	want = Request{
		Cmd:       CmdQuery,
		Tasks:     []string{"abc"},
		QueryArgs: singleDetail(QryDay, "2019-01-01"),
		Combine:   false,
	}
	parseAndCheckExpected(t, want, argQuery, "abc", "--day=2019-01-01")
	// Multiple tasks and days
	want = Request{
		Cmd:   CmdQuery,
		Tasks: []string{"a", "b", "c"},
		// QueryArgs: singleDetail(QryDay, "2019-01-01", "2019-01-02", "2019-01-03"),
		QueryArgs: []QueryDetails{
			QueryDetails{QryDay, "2019-01-01"},
			QueryDetails{QryDay, "2019-01-02"},
			QueryDetails{QryDay, "2019-01-03"},
		},
		Combine: false,
	}
	parseAndCheckExpected(t, want, argQuery, "a,b,c", "--day=2019-01-01,2019-01-02,2019-01-03")
}

func TestQueryRequestMonth(t *testing.T) {
	task := "task"
	want := Request{
		Cmd:     CmdQuery,
		Tasks:   []string{task},
		Combine: false,
	}
	// One explicit month
	want.QueryArgs = singleDetail(QryMonth, "2018-07")
	parseAndCheckExpected(t, want, CmdQuery, task, "--month=2018-07")
	// Multiple explicit months
	want.QueryArgs = []QueryDetails{
		QueryDetails{QryMonth, "2018-12"},
		QueryDetails{QryMonth, "2019-01"},
	}
	parseAndCheckExpected(t, want, CmdQuery, task, "--month=2018-12,2019-01")
	// One relative month
	want.QueryArgs = singleDetail(QryMonth, "2018-05")
	parseAndCheckExpected(t, want, CmdQuery, task, "--months-ago=8")
	// Multiple relative months
	want.QueryArgs = []QueryDetails{
		QueryDetails{QryMonth, "2019-01"},
		QueryDetails{QryMonth, "2018-12"},
		QueryDetails{QryMonth, "2018-10"},
	}
	parseAndCheckExpected(t, want, CmdQuery, task, "--months-ago=0,1,3")
}

func TestQueryRequestYear(t *testing.T) {
	task := "task"
	want := Request{
		Cmd:     CmdQuery,
		Tasks:   []string{task},
		Combine: false,
	}
	// One explicit year
	want.QueryArgs = singleDetail(QryYear, "2016")
	parseAndCheckExpected(t, want, CmdQuery, task, "--year=2016")
	// Several explicit years
	want.QueryArgs = []QueryDetails{
		QueryDetails{QryYear, "2016"},
		QueryDetails{QryYear, "2017"},
		QueryDetails{QryYear, "2018"},
	}
	parseAndCheckExpected(t, want, CmdQuery, task, "--year=2016,2017,2018")
	// One relative year
	want.QueryArgs = singleDetail(QryYear, "2017")
	parseAndCheckExpected(t, want, CmdQuery, task, "--years-ago=2")
	// Multiple relative years
	want.QueryArgs = []QueryDetails{
		QueryDetails{QryYear, "2019"},
		QueryDetails{QryYear, "2018"},
		QueryDetails{QryYear, "2015"},
	}
	parseAndCheckExpected(t, want, CmdQuery, task, "--years-ago=0,1,4")
}

func TestQueryRequestInterval(t *testing.T) {
	task := "foo,bar,baz"
	want := Request{
		Cmd:     CmdQuery,
		Tasks:   []string{"foo", "bar", "baz"},
		Combine: false,
	}
	// Since a given day
	want.QueryArgs = singleDetail(QryBetween, "2017-05-28", today)
	parseAndCheckExpected(t, want, CmdQuery, task, "--since=2017-05-28")
	// Since several days
	want.QueryArgs = []QueryDetails{
		QueryDetails{QryBetween, "1998-12-25", today},
		QueryDetails{QryBetween, "2011-11-11", today},
		QueryDetails{QryBetween, "2018-03-31", today},
	}
	parseAndCheckExpected(t, want, CmdQuery, task,
		"--since=1998-12-25,2011-11-11,2018-03-31")
	// Between two dates
	want.QueryArgs = singleDetail(QryBetween, "2017-01-01", "2018-12-31")
	parseAndCheckExpected(t, want, CmdQuery, task, "--between=2017-01-01,2018-12-31")
}

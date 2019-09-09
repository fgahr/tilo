// Package client describes all client-side operations.
package client

import (
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Reference time for testing.
var now time.Time = time.Date(2019, 1, 8, 12, 0, 0, 0, time.UTC)

const (
	today     = "2019-01-08"
	yesterday = "2019-01-07"
)

func singleDetail(details ...string) []msg.QueryDetails {
	return []msg.QueryDetails{details}
}

func functionExists(fname string) bool {
	handler := server.RequestHandler{}
	nameTargetType := reflect.TypeOf(handler)
	// Methods of this type target a pointer to it
	methodTargetType := reflect.TypeOf(&handler)
	classAndFn := strings.Split(fname, ".")
	if nameTargetType.Name() != classAndFn[0] {
		return false
	}
	_, ok := methodTargetType.MethodByName(classAndFn[1])
	return ok
}

func parseAndCheckExpected(t *testing.T, want msg.Request, args ...string) {
	fname, have, err := msg.ParseRequest(args, now)
	if err != nil {
		t.Error(err)
	} else if !reflect.DeepEqual(want, have) {
		t.Errorf("Wanted %v but got: %v", want, have)
	} else if !functionExists(fname) {
		t.Errorf("Invalid function name: %s", fname)
	}
}

func parseShouldFail(t *testing.T, args ...string) {
	_, _, err := msg.ParseRequest(args, now)
	if err == nil {
		t.Error("Operation expected to fail but succeeded with args:", args)
	}
}

func TestStartRequest(t *testing.T) {
	var want msg.Request
	// start foo
	want = msg.Request{Cmd: msg.CmdStart, Tasks: []string{"foo"}}
	parseAndCheckExpected(t, want, msg.ArgStart, "foo")
	// start bar
	want = msg.Request{Cmd: msg.CmdStart, Tasks: []string{"bar"}}
	parseAndCheckExpected(t, want, msg.ArgStart, "bar")
	// start foo-bar
	want = msg.Request{Cmd: msg.CmdStart, Tasks: []string{"foo-bar"}}
	parseAndCheckExpected(t, want, msg.ArgStart, "foo-bar")

	// start foo,bar
	parseShouldFail(t, msg.ArgStart, "foo,bar")
	parseShouldFail(t, msg.ArgStart, "--cool-task")
	parseShouldFail(t, msg.ArgStart, "cool task")
}

func TestStopRequest(t *testing.T) {
	var want msg.Request
	// stop
	want = msg.Request{Cmd: msg.CmdStop}
	parseAndCheckExpected(t, want, msg.ArgStop)
}

func TestCurrentRequest(t *testing.T) {
	want := msg.Request{Cmd: msg.CmdCurrent}
	parseAndCheckExpected(t, want, msg.ArgCurrent)
}

func TestDiscardRequest(t *testing.T) {
	want := msg.Request{Cmd: msg.CmdAbort}
	parseAndCheckExpected(t, want, msg.ArgAbort)
}

func TestQueryRequestWithoutModifiers(t *testing.T) {
	var want msg.Request
	// Single task, single day
	want = msg.Request{
		Cmd:       msg.CmdQuery,
		Tasks:     []string{"foo"},
		QueryArgs: []msg.QueryDetails{msg.QueryDetails{"day", today}},
		Combine:   false,
	}
	parseAndCheckExpected(t, want, msg.ArgQuery, "foo", msg.PrmToday)
	// Query all tasks
	want = msg.Request{
		Cmd:       msg.CmdQuery,
		Tasks:     []string{msg.TskAllTasks},
		QueryArgs: []msg.QueryDetails{msg.QueryDetails{"day", today}},
		Combine:   false,
	}
	parseAndCheckExpected(t, want, msg.ArgQuery, msg.TskAllTasks, msg.PrmToday)
	// Several tasks, single day
	want = msg.Request{
		Cmd:       msg.CmdQuery,
		Tasks:     []string{"foo", "bar", "baz"},
		QueryArgs: singleDetail(msg.QryDay, yesterday),
		Combine:   false,
	}
	parseAndCheckExpected(t, want, msg.ArgQuery, "foo,bar,baz", msg.PrmYesterday)
	// Single task, several days
	task := "a-simple-task"
	want = msg.Request{
		Cmd:   msg.CmdQuery,
		Tasks: []string{task},
		QueryArgs: []msg.QueryDetails{
			msg.QueryDetails{msg.QryDay, today},
			msg.QueryDetails{msg.QryDay, yesterday},
		},
		Combine: false,
	}
	parseAndCheckExpected(t, want, msg.ArgQuery, task, msg.PrmToday, msg.PrmYesterday)
	// --this-month
	want.QueryArgs = singleDetail(msg.QryMonth, "2019-01")
	parseAndCheckExpected(t, want, msg.ArgQuery, task, msg.PrmThisMonth)
	// --last-month
	want.QueryArgs = singleDetail(msg.QryMonth, "2018-12")
	parseAndCheckExpected(t, want, msg.ArgQuery, task, msg.PrmLastMonth)
	// --last-month --today
	want.QueryArgs = []msg.QueryDetails{
		msg.QueryDetails{msg.QryMonth, "2018-12"},
		msg.QueryDetails{msg.QryDay, today},
	}
	parseAndCheckExpected(t, want, msg.ArgQuery, task, msg.PrmLastMonth, msg.PrmToday)
	// --this-week. Note that `today` (2019-01-08) is a Tuesday
	want.QueryArgs = singleDetail(msg.QryBetween, "2019-01-07", today)
	parseAndCheckExpected(t, want, msg.ArgQuery, task, msg.PrmThisWeek)
	// --last-week. Note that `today` (2019-01-08) is a Tuesday
	want.QueryArgs = singleDetail(msg.QryBetween, "2018-12-31", "2019-01-06")
	parseAndCheckExpected(t, want, msg.ArgQuery, task, msg.PrmLastWeek)
	// --this-year
	want.QueryArgs = singleDetail(msg.QryYear, "2019")
	parseAndCheckExpected(t, want, msg.ArgQuery, task, msg.PrmThisYear)
	// --last-year
	want.QueryArgs = singleDetail(msg.QryYear, "2018")
	parseAndCheckExpected(t, want, msg.ArgQuery, task, msg.PrmLastYear)

	// Invalid tasks
	parseShouldFail(t, msg.ArgQuery, "foo,--bar")
	parseShouldFail(t, msg.ArgQuery, "foo , bar")
	parseShouldFail(t, msg.ArgQuery, "--task")
	// Invalid details
	parseShouldFail(t, msg.ArgQuery, "realtask", "--not-a-real-param")
}

func TestQueryRequestFailInvalidModifiers(t *testing.T) {
	task := "task"
	parseShouldFail(t, msg.ArgQuery, task, "--day=2019-02-31")
	parseShouldFail(t, msg.ArgQuery, task, "--day=2019--2-31")
	parseShouldFail(t, msg.ArgQuery, task, "--day=2019-12-1")
	parseShouldFail(t, msg.ArgQuery, task, "--day=foo")
	parseShouldFail(t, msg.ArgQuery, task, "--month=2018-14")
	parseShouldFail(t, msg.ArgQuery, task, "--month=2013")
	parseShouldFail(t, msg.ArgQuery, task, "--month=foo")
	parseShouldFail(t, msg.ArgQuery, task, "--year=last")
	parseShouldFail(t, msg.ArgQuery, task, "--year=2017-02")
	parseShouldFail(t, msg.ArgQuery, task, "--year=foo")
	parseShouldFail(t, msg.ArgQuery, task, "--between=2017-01-01")
	parseShouldFail(t, msg.ArgQuery, task, "--between=2017-01-01,2018-12-31,2019-04-26")
	parseShouldFail(t, msg.ArgQuery, task, "--between=2017-01-01,foo")
	parseShouldFail(t, msg.ArgQuery, task, "--between=foo")
}

func TestQueryRequestDay(t *testing.T) {
	var want msg.Request
	// Single task and day
	want = msg.Request{
		Cmd:       msg.CmdQuery,
		Tasks:     []string{"abc"},
		QueryArgs: singleDetail(msg.QryDay, "2019-01-01"),
		Combine:   false,
	}
	parseAndCheckExpected(t, want, msg.ArgQuery, "abc", "--day=2019-01-01")
	// Multiple tasks and days
	want = msg.Request{
		Cmd:   msg.CmdQuery,
		Tasks: []string{"a", "b", "c"},
		// QueryArgs: singleDetail(msg.QryDay, "2019-01-01", "2019-01-02", "2019-01-03"),
		QueryArgs: []msg.QueryDetails{
			msg.QueryDetails{msg.QryDay, "2019-01-01"},
			msg.QueryDetails{msg.QryDay, "2019-01-02"},
			msg.QueryDetails{msg.QryDay, "2019-01-03"},
		},
		Combine: false,
	}
	parseAndCheckExpected(t, want, msg.ArgQuery, "a,b,c", "--day=2019-01-01,2019-01-02,2019-01-03")
}

func TestQueryRequestMonth(t *testing.T) {
	task := "task"
	want := msg.Request{
		Cmd:     msg.CmdQuery,
		Tasks:   []string{task},
		Combine: false,
	}
	// One explicit month
	want.QueryArgs = singleDetail(msg.QryMonth, "2018-07")
	parseAndCheckExpected(t, want, msg.CmdQuery, task, "--month=2018-07")
	// Multiple explicit months
	want.QueryArgs = []msg.QueryDetails{
		msg.QueryDetails{msg.QryMonth, "2018-12"},
		msg.QueryDetails{msg.QryMonth, "2019-01"},
	}
	parseAndCheckExpected(t, want, msg.CmdQuery, task, "--month=2018-12,2019-01")
	// One relative month
	want.QueryArgs = singleDetail(msg.QryMonth, "2018-05")
	parseAndCheckExpected(t, want, msg.CmdQuery, task, "--months-ago=8")
	// Multiple relative months
	want.QueryArgs = []msg.QueryDetails{
		msg.QueryDetails{msg.QryMonth, "2019-01"},
		msg.QueryDetails{msg.QryMonth, "2018-12"},
		msg.QueryDetails{msg.QryMonth, "2018-10"},
	}
	parseAndCheckExpected(t, want, msg.CmdQuery, task, "--months-ago=0,1,3")
}

func TestQueryRequestYear(t *testing.T) {
	task := "task"
	want := msg.Request{
		Cmd:     msg.CmdQuery,
		Tasks:   []string{task},
		Combine: false,
	}
	// One explicit year
	want.QueryArgs = singleDetail(msg.QryYear, "2016")
	parseAndCheckExpected(t, want, msg.CmdQuery, task, "--year=2016")
	// Several explicit years
	want.QueryArgs = []msg.QueryDetails{
		msg.QueryDetails{msg.QryYear, "2016"},
		msg.QueryDetails{msg.QryYear, "2017"},
		msg.QueryDetails{msg.QryYear, "2018"},
	}
	parseAndCheckExpected(t, want, msg.CmdQuery, task, "--year=2016,2017,2018")
	// One relative year
	want.QueryArgs = singleDetail(msg.QryYear, "2017")
	parseAndCheckExpected(t, want, msg.CmdQuery, task, "--years-ago=2")
	// Multiple relative years
	want.QueryArgs = []msg.QueryDetails{
		msg.QueryDetails{msg.QryYear, "2019"},
		msg.QueryDetails{msg.QryYear, "2018"},
		msg.QueryDetails{msg.QryYear, "2015"},
	}
	parseAndCheckExpected(t, want, msg.CmdQuery, task, "--years-ago=0,1,4")
}

func TestQueryRequestInterval(t *testing.T) {
	task := "foo,bar,baz"
	want := msg.Request{
		Cmd:     msg.CmdQuery,
		Tasks:   []string{"foo", "bar", "baz"},
		Combine: false,
	}
	// Since a given day
	want.QueryArgs = singleDetail(msg.QryBetween, "2017-05-28", today)
	parseAndCheckExpected(t, want, msg.CmdQuery, task, "--since=2017-05-28")
	// Since several days
	want.QueryArgs = []msg.QueryDetails{
		msg.QueryDetails{msg.QryBetween, "1998-12-25", today},
		msg.QueryDetails{msg.QryBetween, "2011-11-11", today},
		msg.QueryDetails{msg.QryBetween, "2018-03-31", today},
	}
	parseAndCheckExpected(t, want, msg.CmdQuery, task,
		"--since=1998-12-25,2011-11-11,2018-03-31")
	// Between two dates
	want.QueryArgs = singleDetail(msg.QryBetween, "2017-01-01", "2018-12-31")
	parseAndCheckExpected(t, want, msg.CmdQuery, task, "--between=2017-01-01,2018-12-31")
}

package quantifier

import (
	"fmt"
	arg "github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/msg"
	"github.com/pkg/errors"
	"strconv"
	"strings"
	"time"
)

type list struct {
	elem arg.Quantifier
}

func ListOf(elem arg.Quantifier) arg.Quantifier {
	return list{elem}
}

func (lq list) Parse(str string) ([]msg.Quantity, error) {
	qnt := []msg.Quantity{}
	for _, part := range strings.Split(str, ",") {
		nxt, err := lq.elem.Parse(part)
		if err != nil {
			return qnt, err
		}
		qnt = append(qnt, nxt...)
	}
	return qnt, nil
}

func (lq list) DescribeUsage() string {
	return fmt.Sprintf("%s,...", lq.elem.DescribeUsage())
}

type pair struct {
	elem arg.Quantifier
}

func PairOf(elem arg.Quantifier) arg.Quantifier {
	return pair{elem}
}

func (p pair) Parse(str string) ([]msg.Quantity, error) {
	qnt := []msg.Quantity{}
	fields := strings.Split(str, ":")
	if len(fields) != 2 {
		return qnt, errors.New("Not a pair: " + str)
	}
	for _, part := range fields {
		nxt, err := p.elem.Parse(part)
		if err != nil {
			return qnt, err
		}
		qnt = append(qnt, nxt...)
	}
	return qnt, nil
}

func (p pair) DescribeUsage() string {
	return fmt.Sprintf("%s:%[1]s", p.elem.DescribeUsage())
}

type Date struct{}

func (dq Date) Parse(str string) ([]msg.Quantity, error) {
	_, err := time.Parse("2006-01-02", str)
	return arg.SingleQuantity("day", str), err
}

func (dq Date) DescribeUsage() string {
	return "YYYY-MM-DD"
}

type Month struct{}

func (mq Month) Parse(str string) ([]msg.Quantity, error) {
	_, err := time.Parse("2006-01", str)
	return arg.SingleQuantity("month", str), err
}

func (mq Month) DescribeUsage() string {
	return "YYYY-MM"
}

type Year struct{}

func (yq Year) Parse(str string) ([]msg.Quantity, error) {
	_, err := time.Parse("2006", str)
	return arg.SingleQuantity("year", str), err
}

func (yq Year) DescribeUsage() string {
	return "YYYY"
}

type fixedOffset struct {
	now    time.Time
	qType  string
	years  int
	months int
	days   int
}

func (f fixedOffset) Parse(_ string) ([]msg.Quantity, error) {
	return arg.SingleQuantity(f.qType, isoDate(f.now.AddDate(f.years, f.months, f.days))), nil
}

func (f fixedOffset) DescribeUsage() string {
	return ""
}

func FixedDayOffset(now time.Time, days int) arg.Quantifier {
	return fixedOffset{now: now, qType: "day", days: days}
}

func FixedMonthOffset(now time.Time, months int) arg.Quantifier {
	return fixedOffset{now: now, qType: "month", months: months}
}

func FixedYearOffset(now time.Time, years int) arg.Quantifier {
	return fixedOffset{now: now, qType: "year", years: years}
}

// TODO: Combine date offset quantifiers into package-private meta-struct and
// make available via functions?

type DaysAgo struct {
	Now time.Time
}

func (d DaysAgo) Parse(str string) ([]msg.Quantity, error) {
	days, err := strconv.Atoi(str)
	return arg.SingleQuantity("day", isoDate(d.Now.AddDate(0, 0, -days))), err
}

func (d DaysAgo) DescribeUsage() string {
	return "N"
}

type MonthsAgo struct {
	Now time.Time
}

func (m MonthsAgo) Parse(str string) ([]msg.Quantity, error) {
	months, err := strconv.Atoi(str)
	return arg.SingleQuantity("month", isoMonth(m.Now.AddDate(0, -months, 0))), err
}

func (m MonthsAgo) DescribeUsage() string {
	return "N"
}

type YearsAgo struct {
	Now time.Time
}

func (y YearsAgo) Parse(str string) ([]msg.Quantity, error) {
	years, err := strconv.Atoi(str)
	return arg.SingleQuantity("year", isoYear(y.Now.AddDate(0, 0, -years))), err
}

func (y YearsAgo) DescribeUsage() string {
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

// Format as yyyy.
func isoYear(t time.Time) string {
	return t.Format("2006")
}

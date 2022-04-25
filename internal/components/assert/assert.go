package assert

import (
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

type operationType string

const (
	Noop       operationType = "noop"
	Equal      operationType = "eq"
	NotEqual   operationType = "ne"
	GreatEqual operationType = "ge"
	GreatThan  operationType = "gt"
	LessEqual  operationType = "le"
	LessThan   operationType = "lt"
	Empty      operationType = "empty"
	NotEmpty   operationType = "notEmpty"
)

var assertRegister map[operationType]Interface

// initRegister init assert func
func initRegister() {
	assertRegister = make(map[operationType]Interface)
	assertRegister[Noop] = new(noopAssert)
	assertRegister[Equal] = new(equalAssert)
	assertRegister[NotEqual] = new(notEqualAssert)
	assertRegister[GreatEqual] = new(greatEqualAssert)
	assertRegister[GreatThan] = new(greatThanAssert)
	assertRegister[LessEqual] = new(lessEqualAssert)
	assertRegister[LessThan] = new(lessThanAssert)
	assertRegister[Empty] = new(emptyAssert)
	assertRegister[NotEmpty] = new(notEmptyAssert)
}

type Interface interface {
	Assert(expectedValue, actualValue string) error
}

type expression struct {
	operation     operationType
	expectedValue string
}

var expressionCache sync.Map

func init() {
	expressionCache = sync.Map{}
	expressionCache.Store("noop", &expression{operation: Noop})
	initRegister()
}

func parseExpression(exp string) *expression {
	if e, ok := expressionCache.Load(exp); ok {
		return e.(*expression)
	}

	exp = strings.TrimSpace(exp)
	if exp == "" {
		e, _ := expressionCache.Load("noop")
		return e.(*expression)
	}
	expSlice := strings.Split(exp, " ")
	op := expSlice[0]
	value := op
	if len(expSlice) == 2 {
		value = expSlice[1]
	}

	var e *expression
	operation := operationType(op)
	switch operation {
	case Noop, Equal, NotEqual, Empty, NotEmpty, GreatEqual, GreatThan, LessEqual, LessThan:
		e = &expression{operation: operation, expectedValue: value}
	default:
		e = &expression{operation: Equal, expectedValue: exp}
	}

	expressionCache.Store(exp, e)
	return e
}

func ValueAssert(desc, expectedExpress, actualValue string) error {
	exp := parseExpression(expectedExpress)
	assert := assertRegister[exp.operation]
	err := assert.Assert(exp.expectedValue, actualValue)
	if err != nil {
		return errors.Wrap(err, desc)
	}
	return nil
}

type noopAssert struct{}

func (*noopAssert) Assert(expectedValue, actualValue string) error {
	return nil
}

type equalAssert struct{}

func (*equalAssert) Assert(expectedValue, actualValue string) error {
	if expectedValue != actualValue {
		return errors.Errorf("--expectedValue => %s, ++actualValue => %s", expectedValue, actualValue)
	}
	return nil
}

type notEqualAssert struct{}

func (*notEqualAssert) Assert(expectedValue, actualValue string) error {
	if expectedValue == actualValue {
		return errors.Errorf("--expectedValue => notEqual %s, ++actualValue => %s", expectedValue, actualValue)
	}
	return nil
}

type greatThanAssert struct{}

func (*greatThanAssert) Assert(expectedValue, actualValue string) error {
	expected, err := strconv.ParseFloat(expectedValue, 64)
	if err != nil {
		return errors.Wrap(err, "parse expectedValue error")
	}
	actual, err := strconv.ParseFloat(actualValue, 64)
	if err != nil {
		return errors.Wrap(err, "parse actual error")
	}

	if expected >= actual {
		return errors.Errorf("--expectedValue => gt %s, ++actualValue => %s", expectedValue, actualValue)
	}
	return nil
}

type greatEqualAssert struct{}

func (*greatEqualAssert) Assert(expectedValue, actualValue string) error {
	expected, err := strconv.ParseFloat(expectedValue, 64)
	if err != nil {
		return errors.Wrap(err, "parse expectedValue error")
	}
	actual, err := strconv.ParseFloat(actualValue, 64)
	if err != nil {
		return errors.Wrap(err, "parse actual error")
	}

	if expected > actual {
		return errors.Errorf("--expectedValue => ge %s, ++actualValue => %s", expectedValue, actualValue)
	}
	return nil
}

type lessThanAssert struct{}

func (*lessThanAssert) Assert(expectedValue, actualValue string) error {
	expected, err := strconv.ParseFloat(expectedValue, 64)
	if err != nil {
		return errors.Wrap(err, "parse expectedValue error")
	}
	actual, err := strconv.ParseFloat(actualValue, 64)
	if err != nil {
		return errors.Wrap(err, "parse actual error")
	}

	if expected <= actual {
		return errors.Errorf("--expectedValue => lt %s, ++actualValue => %s", expectedValue, actualValue)
	}
	return nil
}

type lessEqualAssert struct{}

func (*lessEqualAssert) Assert(expectedValue, actualValue string) error {
	expected, err := strconv.ParseFloat(expectedValue, 64)
	if err != nil {
		return errors.Wrap(err, "parse expectedValue error")
	}
	actual, err := strconv.ParseFloat(actualValue, 64)
	if err != nil {
		return errors.Wrap(err, "parse actual error")
	}

	if expected < actual {
		return errors.Errorf("--expectedValue => le %s, ++actualValue => %s", expectedValue, actualValue)
	}
	return nil
}

type emptyAssert struct{}

func (*emptyAssert) Assert(_, actualValue string) error {
	if actualValue != "" {
		return errors.Errorf("--expectedValue => %s, ++actualValue => %s", "empty", actualValue)
	}
	return nil
}

type notEmptyAssert struct{}

func (*notEmptyAssert) Assert(_, actualValue string) error {
	if actualValue == "" {
		return errors.Errorf("--expectedValue => %s, ++actualValue => %s", "notEmpty", actualValue)
	}
	return nil
}

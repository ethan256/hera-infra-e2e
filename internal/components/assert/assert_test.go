package assert

import (
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
)

type assertCase struct {
	Desc     string `json:"desc"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	WantFail bool   `json:"wantFail"`
}

var caseStr = `
[
{
	"desc": "case 01",
	"expected": "notEmpty",
	"actual": "qwer"
},
{
	"desc": "case 02",
	"expected": "empty",
	"actual": ""
},
{
	"desc": "case 03",
	"expected": "lt 0",
	"actual": "-2"
},
{
	"desc": "case 04",
	"expected": "le 0",
	"actual": "-2"
},
{
	"desc": "case 05",
	"expected": "gt 0",
	"actual": "2"
},
{
	"desc": "case 06",
	"expected": "ge 0",
	"actual": "2"
},
{
	"desc": "case 07",
	"expected": "eq 1",
	"actual": "1"
},
{
	"desc": "case 08",
	"expected": "ne 0",
	"actual": "2"
},
{
	"desc": "case 09",
	"expected": "regexp ^v(([0-9]|([1-9]([0-9]*))).){2}([0-9]|([1-9]([0-9]*)))([-](([0-9A-Za-z]|([1-9A-Za-z]([0-9A-Za-z]*)))[.]){0,}([0-9A-Za-z]|([1-9A-Za-z]([0-9A-Za-z]*)))){0,1}([+](([0-9A-Za-z]{1,})[.]){0,}([0-9A-Za-z]{1,})){0,1}$",
	"actual": "v1.2.3"
},
{
	"desc": "case 10",
	"expected": "regexp ^v(([0-9]|([1-9]([0-9]*))).){2}([0-9]|([1-9]([0-9]*)))([-](([0-9A-Za-z]|([1-9A-Za-z]([0-9A-Za-z]*)))[.]){0,}([0-9A-Za-z]|([1-9A-Za-z]([0-9A-Za-z]*)))){0,1}([+](([0-9A-Za-z]{1,})[.]){0,}([0-9A-Za-z]{1,})){0,1}$",
	"actual": "v1.2.3-abc"
},
{
	"desc": "case 11",
	"expected": "regexp ^v(([0-9]|([1-9]([0-9]*))).){2}([0-9]|([1-9]([0-9]*)))([-](([0-9A-Za-z]|([1-9A-Za-z]([0-9A-Za-z]*)))[.]){0,}([0-9A-Za-z]|([1-9A-Za-z]([0-9A-Za-z]*)))){0,1}([+](([0-9A-Za-z]{1,})[.]){0,}([0-9A-Za-z]{1,})){0,1}$",
	"actual": "v1.22.0"
},
{
	"desc": "case 12",
	"expected": "regexp ^v(([0-9]|([1-9]([0-9]*))).){2}([0-9]|([1-9]([0-9]*)))([-](([0-9A-Za-z]|([1-9A-Za-z]([0-9A-Za-z]*)))[.]){0,}([0-9A-Za-z]|([1-9A-Za-z]([0-9A-Za-z]*)))){0,1}([+](([0-9A-Za-z]{1,})[.]){0,}([0-9A-Za-z]{1,})){0,1}$",
	"actual": "v1.22.abc",
	"wantFail": true
}
]
`

func TestValueAssert(t *testing.T) {
	cases, err := buildAssertCase()
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cases {
		if err := runTestCase(c); err != nil && !c.WantFail {
			t.Errorf("case: %s, error: %s", c.Desc, err.Error())
		}
	}
}

func BenchmarkValueAssert(b *testing.B) {
	cases, err := buildAssertCase()
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		for _, c := range cases {
			if err := runTestCase(c); err != nil {
				b.Errorf("case: %s, error: %s", c.Desc, err.Error())
			}
		}
	}
}

func BenchmarkValueAssertWithoutCache(b *testing.B) {
	cases, err := buildAssertCase()
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		for _, c := range cases {
			if err := runTestCaseWithoutCache(c); err != nil {
				b.Errorf("case: %s, error: %s", c.Desc, err.Error())
			}
		}
	}
}

func runTestCase(c *assertCase) error {
	return ValueAssert(c.Desc, c.Expected, c.Actual)
}

func runTestCaseWithoutCache(c *assertCase) error {
	return valueAssertWithoutCache(c.Desc, c.Expected, c.Actual)
}

func buildAssertCase() ([]*assertCase, error) {
	var cases []*assertCase
	if err := json.Unmarshal([]byte(caseStr), &cases); err != nil {
		return nil, err
	}
	return cases, nil
}

func valueAssertWithoutCache(desc, expectedExpress, actualValue string) error {
	exp := parseExpression(expectedExpress)
	assert := assertRegister[exp.operation]
	err := assert.Assert(exp.expectedValue, actualValue)
	if err != nil {
		return errors.Wrap(err, desc)
	}
	return nil
}

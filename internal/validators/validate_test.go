package validators_test

import (
	"testing"

	"github.com/netresearch/ldap-selfservice-password-changer/internal/validators"
)

type TestCase struct {
	Input    string
	Arg      uint
	Expected bool
}

func TestMinNumbersInString(t *testing.T) {
	cases := []TestCase{
		{
			Input:    "asdf1f",
			Arg:      1,
			Expected: true,
		},
		{
			Input:    "asdf1f",
			Arg:      2,
			Expected: false,
		},
	}

	for _, c := range cases {
		actual := validators.MinNumbersInString(c.Input, c.Arg)
		if actual != c.Expected {
			t.Errorf("expected %t, got %t", c.Expected, actual)
		}
	}
}

func TestMinSymbolsInString(t *testing.T) {
	cases := []TestCase{
		{
			Input:    "asdf!f",
			Arg:      1,
			Expected: true,
		},
		{
			Input:    "asdf!f",
			Arg:      2,
			Expected: false,
		},
		{
			Input:    "a!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~b",
			Arg:      32,
			Expected: true,
		},
	}

	for _, c := range cases {
		actual := validators.MinSymbolsInString(c.Input, c.Arg)
		if actual != c.Expected {
			t.Errorf("expected %t, got %t", c.Expected, actual)
		}
	}
}

func TestMinUppercaseLettersInString(t *testing.T) {
	cases := []TestCase{
		{
			Input:    "asdfF",
			Arg:      1,
			Expected: true,
		},
		{
			Input:    "asdfF",
			Arg:      2,
			Expected: false,
		},
	}

	for _, c := range cases {
		actual := validators.MinUppercaseLettersInString(c.Input, c.Arg)
		if actual != c.Expected {
			t.Errorf("expected %t, got %t", c.Expected, actual)
		}
	}
}

func TestMinLowercaseLettersInString(t *testing.T) {
	cases := []TestCase{
		{
			Input:    "asdfF",
			Arg:      1,
			Expected: true,
		},
		{
			Input:    "asdfF",
			Arg:      5,
			Expected: false,
		},
	}

	for _, c := range cases {
		actual := validators.MinLowercaseLettersInString(c.Input, c.Arg)
		if actual != c.Expected {
			t.Errorf("expected %t, got %t", c.Expected, actual)
		}
	}
}

package nib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeTempFilename(t *testing.T) {
	type testCase struct {
		reference string
		summary   string
		expected  string
	}

	testCases := []testCase{
		{
			"1a2b3c4d",
			`& &`,
			`nib.*.1a2b3c4d-.md`,
		},
		{
			"1A2B3C4D",
			`J's $100, != €100`,
			`nib.*.1a2b3c4d-js-100-100.md`,
		},
		{
			"99",
			`A simple summary!`,
			`nib.*.99-a-simple-summary.md`,
		},
		{
			"1",
			`& that's that.`,
			`nib.*.1-thats-that.md`,
		},
	}

	for _, tc := range testCases {
		tf := MakeTempFilename(tc.reference, tc.summary, "md")

		assert.Equal(t, tc.expected, tf)

		f, err := os.CreateTemp(t.TempDir(), tf)
		assert.NoError(t, err)
		assert.NoError(t, f.Close())
		assert.NoError(t, os.Remove(f.Name()))
	}
}

func TestStrSliceContainsAll(t *testing.T) {
	type testCase struct {
		subset   []string
		superset []string
		expected bool
	}

	testCases := []testCase{
		{
			[]string{},
			[]string{},
			true,
		},
		{
			[]string{"one"},
			[]string{"one"},
			true,
		},
		{
			[]string{"one"},
			[]string{"two"},
			false,
		},
		{
			[]string{"one"},
			[]string{},
			false,
		},
		{
			[]string{"one"},
			[]string{"one", "two"},
			true,
		},
		{
			[]string{"one", "two"},
			[]string{"one", "two"},
			true,
		},
		{
			[]string{"two", "one"},
			[]string{"three", "one", "two"},
			true,
		},
		{
			[]string{"apple", "two", "one"},
			[]string{"three", "one", "two"},
			false,
		},
		{
			[]string{},
			[]string{"three", "one", "two"},
			true,
		},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.expected, StrSliceContainsAll(tc.subset, tc.superset))
	}
}

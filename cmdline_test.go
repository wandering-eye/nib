package nib

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseQuery(t *testing.T) {
	type testCase struct {
		input    []string
		expected Query
	}

	tests := []testCase{
		{
			[]string{"add", "have", "an", "adventure"},
			Query{
				Cmd:               "add",
				References:        nil,
				Tags:              nil,
				AntiTags:          nil,
				Project:           "",
				AntiProjects:      nil,
				TemplateReference: "",
				Text:              "have an adventure",
				IgnoreContext:     false,
				Note:              "",
			},
		},
		{
			[]string{"add", "+x", "-y", "have", "an", "adventure"},
			Query{
				Cmd:               "add",
				References:        nil,
				Tags:              []string{"x"},
				AntiTags:          []string{"y"},
				Project:           "",
				AntiProjects:      nil,
				TemplateReference: "",
				Text:              "have an adventure",
				IgnoreContext:     false,
				Note:              "",
			},
		},
		{
			[]string{"add", "smile", "/"},
			Query{
				Cmd:               "add",
				References:        nil,
				Tags:              nil,
				AntiTags:          nil,
				Project:           "",
				AntiProjects:      nil,
				TemplateReference: "",
				Text:              "smile",
				IgnoreContext:     false,
				Note:              "",
			},
		},

		{
			[]string{"add", "floss", "project:p", "+health", "/", "every  day"},
			Query{
				Cmd:               "add",
				References:        nil,
				Tags:              []string{"health"},
				AntiTags:          nil,
				Project:           "p",
				AntiProjects:      nil,
				TemplateReference: "",
				Text:              "floss",
				IgnoreContext:     false,
				Note:              "every  day",
			},
		},
		{
			[]string{"1a2b", "modify", "+project:p", "-project:x", "-fun"},
			Query{
				Cmd:               "modify",
				References:        []string{"1a2b"},
				Tags:              nil,
				AntiTags:          []string{"fun"},
				Project:           "p",
				AntiProjects:      []string{"x"},
				TemplateReference: "",
				Text:              "",
				IgnoreContext:     false,
				Note:              "",
			},
		},
		{
			[]string{"--", "show-resolved"},
			Query{
				Cmd:               "show-resolved",
				References:        nil,
				Tags:              nil,
				AntiTags:          nil,
				Project:           "",
				AntiProjects:      nil,
				TemplateReference: "",
				Text:              "",
				IgnoreContext:     true,
				Note:              "",
			},
		},
		// first priority should have precedence, subsequent priorities should
		// just be part of the description
		// see https://github.com/wandering-eye/nib/issues/120 for context
		{
			[]string{"add", "P1", "P2", "P3"},
			Query{
				Cmd:               "add",
				References:        nil,
				Tags:              nil,
				AntiTags:          nil,
				Project:           "",
				AntiProjects:      nil,
				Priority:          PRIORITY_HIGH,
				TemplateReference: "",
				Text:              "P2 P3",
				IgnoreContext:     false,
				Note:              "",
			},
		},
		// same for projects, for consistency
		{
			[]string{"add", "project:foo", "project:bar"},
			Query{
				Cmd:               "add",
				References:        nil,
				Tags:              nil,
				AntiTags:          nil,
				Project:           "foo",
				AntiProjects:      nil,
				TemplateReference: "",
				Text:              "project:bar",
				IgnoreContext:     false,
				Note:              "",
			},
		},
		{
			[]string{"add", "My", "Task", "template:1A2b", "/", "Test", "Note"},
			Query{
				Cmd:               "add",
				References:        nil,
				Tags:              nil,
				AntiTags:          nil,
				Project:           "",
				AntiProjects:      nil,
				TemplateReference: "1a2b",
				Text:              "My Task",
				IgnoreContext:     false,
				Note:              "Test Note",
			},
		},
		// a creating command takes no reference, so a word that only looks
		// like one stays in the summary
		{
			[]string{"add", "cafe", "2024", "order"},
			Query{
				Cmd:  "add",
				Text: "cafe 2024 order",
			},
		},
		{
			[]string{"log", "deadbeef"},
			Query{
				Cmd:  "log",
				Text: "deadbeef",
			},
		},
		// references may be pasted UUIDs, in any case, before or after the
		// command; a leading hyphen is always an anti-tag
		{
			[]string{"6F1E2D3C-aaaa-4bbb-8ccc-1234567890ab", "cafe", "done", "-cafe", "beef"},
			Query{
				Cmd:        "done",
				References: []string{"6F1E2D3C-aaaa-4bbb-8ccc-1234567890ab", "cafe"},
				AntiTags:   []string{"cafe"},
				Text:       "beef",
			},
		},
		// "--" is made only of hyphens and is never a reference; like any
		// other operator it ends the references
		{
			[]string{"abcd", "--", "beef"},
			Query{
				References:    []string{"abcd"},
				Text:          "beef",
				IgnoreContext: true,
			},
		},
	} // end test cases

	for i, tc := range tests {
		description := strings.Join(tc.input, " ")

		t.Run(fmt.Sprintf("test %v: %s", i, description), func(t *testing.T) {
			t.Parallel()

			actual := ParseQuery(tc.input...)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

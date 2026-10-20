package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Earlier versions wrote whatever offset the machine happened to be in, so one store
// ended up holding a mix of Z and -07:00 and sorting it as text put records in
// the wrong order. What is stored is UTC; local time is input and display.
func TestStoredTimestampsAreUTC(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	restore := setEnv("TZ", "America/Los_Angeles")
	defer restore()

	program := testCmd(repo)

	output, exiterr, success := program("add", "with a due date", "due:tomorrow")
	assertProgramResult(test, output, exiterr, success)

	output, exiterr, success = program("add", "to be resolved")
	assertProgramResult(test, output, exiterr, success)

	output, exiterr, success = program("done", referenceFor(test, repo, "to be resolved"))
	assertProgramResult(test, output, exiterr, success)

	offset := regexp.MustCompile(`\d{2}:\d{2}:\d{2}(\.\d+)?[+-]\d{2}:\d{2}`)

	var checked int

	for _, status := range []string{"pending", "resolved"} {
		matches, err := filepath.Glob(filepath.Join(repo, status, "*.yml"))
		assert.NoError(test, err)

		for _, match := range matches {
			stored, err := os.ReadFile(match)
			assert.NoError(test, err)

			assert.NotRegexp(test, offset, string(stored),
				"%s carries a numeric offset; stored instants must be UTC", match)

			for _, line := range strings.Split(string(stored), "\n") {
				for _, field := range []string{"created:", "resolved:", "due:"} {
					if strings.HasPrefix(line, field) {
						assert.True(test, strings.HasSuffix(strings.TrimSpace(line), "Z"),
							"%s must end in Z, got %q", field, line)

						checked++
					}
				}
			}
		}
	}

	assert.Greater(test, checked, 2, "expected several timestamps to inspect")
}

// The same instant must land on disk identically whatever zone the machine is
// in, otherwise two clones disagree about ordering.
func TestStorageDoesNotDependOnTheMachineTimezone(test *testing.T) {
	west, cleanupWest := makeRepo(test)
	defer cleanupWest()

	east, cleanupEast := makeRepo(test)
	defer cleanupEast()

	restore := setEnv("TZ", "America/Los_Angeles")
	output, exiterr, success := testCmd(west)("add", "same instant")
	assertProgramResult(test, output, exiterr, success)
	restore()

	restore = setEnv("TZ", "Europe/Berlin")
	output, exiterr, success = testCmd(east)("add", "same instant")
	assertProgramResult(test, output, exiterr, success)
	restore()

	for _, repo := range []string{west, east} {
		matches, err := filepath.Glob(filepath.Join(repo, "pending", "*.yml"))
		assert.NoError(test, err)
		assert.Len(test, matches, 1)

		stored, err := os.ReadFile(matches[0])
		assert.NoError(test, err)
		assert.Contains(test, string(stored), "Z\n", "stored instants are UTC in every zone")
	}
}

// Storage is UTC, so rendering has to convert back before deciding which day a
// due date falls on. Comparing a UTC instant against a local now put every due
// date a day out for anyone east of UTC: "tomorrow" rendered as "today".
func TestDueDateRendersInTheReadersDay(test *testing.T) {
	for _, zone := range []string{"Asia/Tokyo", "Pacific/Kiritimati", "America/Los_Angeles", "UTC"} {
		repo, cleanup := makeRepo(test)

		restore := setEnv("TZ", zone)

		program := testTableCmd(repo)

		// Two tasks: a single match renders the detail view, which prints the
		// raw instant and never reaches the humanised Due column.
		output, exiterr, success := program("add", "lands tomorrow", "due:tomorrow")
		assertProgramResult(test, output, exiterr, success)

		output, exiterr, success = program("add", "no due date")
		assertProgramResult(test, output, exiterr, success)

		rendered, _, _ := program("next")

		assert.Contains(test, string(rendered), "tomorrow",
			"in %s a task due tomorrow must render as tomorrow, got: %s", zone, rendered)
		assert.NotContains(test, string(rendered), "today",
			"in %s it must not render as today", zone)

		restore()
		cleanup()
	}
}

package integration

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoteAppendTextWithoutTTY(t *testing.T) {
	repo, cleanup := makeRepo(t)
	defer cleanup()

	program := testCmd(repo)

	output, exiterr, success := program("add", "test task")
	assertProgramResult(t, output, exiterr, success)

	_, exiterr, success = program("note", referenceFor(t, repo, "test task"), "first line")
	assertProgramResult(t, nil, exiterr, success)

	output, exiterr, success = program("show-open")
	assertProgramResult(t, output, exiterr, success)

	tasks := unmarshalTaskArray(t, output)
	require.Len(t, tasks, 1)
	assert.Equal(t, "first line", tasks[0].Notes)

	_, exiterr, success = program("note", referenceFor(t, repo, "test task"), "second line")
	assertProgramResult(t, nil, exiterr, success)

	output, exiterr, success = program("show-open")
	assertProgramResult(t, output, exiterr, success)

	tasks = unmarshalTaskArray(t, output)
	require.Len(t, tasks, 1)
	assert.Equal(t, "first line\nsecond line", tasks[0].Notes)
}

// Notes are for learnings after the fact too, so a resolved task takes one
// without being reopened, moved, or restamped.
func TestNoteOnAResolvedTaskLeavesItResolved(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	program := testCmd(repo)

	output, exiterr, success := program("add", "finished task")
	assertProgramResult(test, output, exiterr, success)

	reference := referenceFor(test, repo, "finished task")

	output, exiterr, success = program(reference, "done")
	assertProgramResult(test, output, exiterr, success)

	records, err := filepath.Glob(filepath.Join(repo, "resolved", "*.yml"))
	require.NoError(test, err)
	require.Len(test, records, 1)

	resolvedLine := regexp.MustCompile(`(?m)^resolved: .*$`)
	before, err := os.ReadFile(records[0])
	require.NoError(test, err)
	resolvedAt := resolvedLine.Find(before)
	require.NotEmpty(test, resolvedAt)

	for _, arguments := range [][]string{
		{reference, "note", "learned afterwards"},
		{"note", reference, "and later still"},
	} {
		commits := runGit(test, "-C", repo, "rev-list", "--count", "HEAD")

		output, exiterr, success = program(arguments...)
		assertProgramResult(test, output, exiterr, success)

		assert.Equal(test, increment(test, commits),
			runGit(test, "-C", repo, "rev-list", "--count", "HEAD"), "one note is one commit")
		assert.Equal(test, "Edit note "+reference+": finished task",
			runGit(test, "-C", repo, "log", "-1", "--format=%s"))
	}

	after, err := os.ReadFile(records[0])
	require.NoError(test, err, "the record stays in resolved/")
	assert.Contains(test, string(after), "learned afterwards")
	assert.Contains(test, string(after), "and later still")
	assert.Equal(test, string(resolvedAt), string(resolvedLine.Find(after)))

	open, err := filepath.Glob(filepath.Join(repo, "pending", "*.yml"))
	require.NoError(test, err)
	assert.Empty(test, open)

	assert.Empty(test, runGit(test, "-C", repo, "status", "--porcelain", "--untracked-files=no"))
}

// note looks at every status, so an open and a resolved task sharing a prefix
// are both candidates.
func TestNoteOnAPrefixSharedWithAResolvedTaskIsAmbiguous(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	writeRecord(test, repo, "pending", "aaaa1111-0000-4000-8000-000000000001", "open twin")
	writeRecord(test, repo, "resolved", "aaaa1111-0000-4000-8000-000000000002", "resolved twin")

	_, exiterr, _ := testCmd(repo)("note", "aaaa1111", "which one")
	require.NotNil(test, exiterr)
	assert.Equal(test, 5, exiterr.ExitCode())
	assert.Contains(
		test,
		string(exiterr.Stderr),
		"aaaa1111-0000-4000-8000-000000000001: open twin\n",
	)
	assert.Contains(
		test,
		string(exiterr.Stderr),
		"aaaa1111-0000-4000-8000-000000000002: resolved twin (resolved)",
	)
}

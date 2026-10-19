package integration

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Output shape used to depend on whether stdout was a terminal, which is how
// `note <id> <text>` became a silent read when piped, because output shape was
// chosen by inspecting stdout. Nothing may inspect it now, so the same command
// has to produce the same bytes down a pipe as into a file on disk.
func TestOutputDoesNotDependOnDestination(test *testing.T) {
	repo, cleanup := makeDstaskRepo(test)
	defer cleanup()

	table := testTableCmd(repo)

	for _, summary := range []string{"first entry", "second entry"} {
		output, exiterr, success := table("add", summary, "project:alpha")
		assertProgramResult(test, output, exiterr, success)
	}

	// A pipe: exec.Cmd.Output attaches one.
	piped, _, _ := table("next")

	// A regular file, which is what a shell redirect gives the process.
	file, err := os.CreateTemp("", "nib-redirect-")
	assert.NoError(test, err)

	defer func() { _ = os.Remove(file.Name()) }()

	command := exec.Command(binaryPath(), "next")
	command.Env = repoEnvironment(repo)
	command.Stdout = file
	assert.NoError(test, command.Run())
	assert.NoError(test, file.Close())

	redirected, err := os.ReadFile(file.Name())
	assert.NoError(test, err)

	assert.Equal(test, string(piped), string(redirected),
		"output differs between a pipe and a file")
	assert.Contains(test, string(piped), "first entry")
	assert.False(test, bytes.HasPrefix(bytes.TrimSpace(piped), []byte("[")),
		"the table is the default; JSON must be asked for")
}

// JSON is opt-in. dstask emitted it automatically whenever stdout was not a
// terminal, so a consumer could not tell a table from a machine format.
func TestJSONIsExplicit(test *testing.T) {
	repo, cleanup := makeDstaskRepo(test)
	defer cleanup()

	program := testTableCmd(repo)
	output, exiterr, success := program("add", "explicit json")
	assertProgramResult(test, output, exiterr, success)

	asJSON, _, _ := program("next", "--json")
	assert.True(test, bytes.HasPrefix(bytes.TrimSpace(asJSON), []byte("[")))

	asTable, _, _ := program("next")
	assert.False(test, bytes.HasPrefix(bytes.TrimSpace(asTable), []byte("[")))
}

// Colour is declared, never detected. NO_COLOR has to reach the escapes even
// though nothing knows what stdout is attached to.
func TestNoColorSuppressesEscapes(test *testing.T) {
	repo, cleanup := makeDstaskRepo(test)
	defer cleanup()

	program := testTableCmd(repo)
	output, exiterr, success := program("add", "coloured row", "P1")
	assertProgramResult(test, output, exiterr, success)

	coloured, _, _ := program("next")
	assert.Contains(test, string(coloured), "\x1b[", "colour is on by default")

	unset := setEnv("NO_COLOR", "1")
	defer unset()

	plain, _, _ := program("next")
	assert.NotContains(test, string(plain), "\x1b[", "NO_COLOR must remove every escape")
}

// An id names one task, so naming nothing is an error. A filter matching
// nothing is a legitimate empty answer and stays 0.
func TestMissingIDExitsNoMatchAndFilterDoesNot(test *testing.T) {
	repo, cleanup := makeDstaskRepo(test)
	defer cleanup()

	program := testTableCmd(repo)
	output, exiterr, success := program("add", "present")
	assertProgramResult(test, output, exiterr, success)

	_, exiterr, _ = program("9999")
	assert.NotNil(test, exiterr, "a missing id must fail")
	assert.Equal(test, 3, exiterr.ExitCode(), "a missing id is exit 3")

	_, exiterr, success = program("project:nothingmatchesthis")
	assert.Nil(test, exiterr, "a filter matching nothing is not an error")
	assert.True(test, success)
}

// dstask skipped the confirmation when stdout was not a terminal and modified
// every task in the context regardless. Declining must leave them alone.
func TestBulkModifyWithoutConfirmationChangesNothing(test *testing.T) {
	repo, cleanup := makeDstaskRepo(test)
	defer cleanup()

	program := testCmd(repo)
	output, exiterr, success := program("add", "untouched", "project:alpha")
	assertProgramResult(test, output, exiterr, success)

	_, exiterr, _ = runBinary(repo, "modify", "P1")
	assert.NotNil(test, exiterr, "an unconfirmed bulk change must not succeed")
	assert.Equal(test, 4, exiterr.ExitCode(), "declining is a no-op, exit 4")

	output, _, _ = program("next")
	tasks := unmarshalTaskArray(test, output)
	assert.Equal(test, "P2", tasks[0].Priority, "priority must be unchanged")

	confirmed := testConfirmingCmd(repo)
	output, exiterr, success = confirmed("modify", "P1")
	assertProgramResult(test, output, exiterr, success)

	output, _, _ = program("next")
	tasks = unmarshalTaskArray(test, output)
	assert.Equal(test, "P1", tasks[0].Priority, "confirming applies the change")
}

// `note <id>` with no text must never print the note. Falling through to a
// read is the defect this project is named for.
func TestNoteWithoutTextDoesNotPrintTheNote(test *testing.T) {
	repo, cleanup := makeDstaskRepo(test)
	defer cleanup()

	program := testTableCmd(repo)
	output, exiterr, success := program("add", "has a note")
	assertProgramResult(test, output, exiterr, success)

	output, exiterr, success = program("note", "1", "the note body")
	assertProgramResult(test, output, exiterr, success)

	unset := setEnv("EDITOR", "false")
	defer unset()

	printed, exiterr, _ := runBinary(repo, "note", "1")
	assert.NotNil(test, exiterr, "no text means the editor, and a failing editor fails")
	assert.False(test, strings.Contains(string(printed), "the note body"),
		"note must not degrade into printing the note")
}

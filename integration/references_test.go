package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runGit(test *testing.T, arguments ...string) string {
	test.Helper()

	output, err := exec.Command("git", arguments...).CombinedOutput()
	require.NoError(test, err, "git %v: %s", arguments, output)

	return strings.TrimSpace(string(output))
}

// writeRecord puts a task on disk directly, so a test can choose its UUID.
func writeRecord(test *testing.T, repo, status, uuid, summary string) {
	test.Helper()

	require.NoError(test, os.MkdirAll(filepath.Join(repo, status), 0o700))

	record := "summary: " + summary + "\npriority: P2\ncreated: 2026-01-01T00:00:00Z\n"
	require.NoError(
		test,
		os.WriteFile(filepath.Join(repo, status, uuid+".yml"), []byte(record), 0o600),
	)
}

func TestAddReportsTheShortUUID(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	output, exiterr, success := testTableCmd(repo)("add", "named by uuid")
	assertProgramResult(test, output, exiterr, success)

	reference := referenceFor(test, repo, "named by uuid")
	assert.Contains(test, string(output), "Added "+reference+": named by uuid")

	assert.Equal(
		test,
		"Added "+reference+": named by uuid",
		runGit(test, "-C", repo, "log", "-1", "--format=%s"),
	)

	records, err := filepath.Glob(filepath.Join(repo, "pending", "*.yml"))
	require.NoError(test, err)
	require.Len(test, records, 1)

	stored, err := os.ReadFile(records[0])
	require.NoError(test, err)
	assert.NotContains(test, string(stored), "key:", "a task's identity is its UUID")

	_, err = os.Stat(filepath.Join(repo, "next-key"))
	assert.True(test, os.IsNotExist(err), "nothing is allocated, so there is no counter")
}

// A prefix shared by two tasks is refused with both listed, never guessed.
func TestReferenceResolutionExitCodes(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	writeRecord(test, repo, "pending", "aaaa1111-0000-4000-8000-000000000001", "first twin")
	writeRecord(test, repo, "pending", "aaaa1111-0000-4000-8000-000000000002", "second twin")
	writeRecord(test, repo, "pending", "bbbb2222-0000-4000-8000-000000000003", "loner")

	program := testCmd(repo)

	output, exiterr, success := program("bbbb")
	assertProgramResult(test, output, exiterr, success)
	tasks := unmarshalTaskArray(test, output)
	require.Len(test, tasks, 1)
	assert.Equal(test, "loner", tasks[0].Summary)
	assert.Equal(
		test,
		"bbbb2222-0000-4000-8000-000000000003",
		tasks[0].UUID,
		"JSON carries the full UUID",
	)

	output, exiterr, success = program("aaaa1111-0000-4000-8000-000000000002")
	assertProgramResult(test, output, exiterr, success)
	assert.Equal(test, "second twin", unmarshalTaskArray(test, output)[0].Summary)

	_, exiterr, _ = program("aaaa1111", "done")
	require.NotNil(test, exiterr, "an ambiguous reference must fail")
	assert.Equal(test, 5, exiterr.ExitCode(), "an ambiguous reference is exit 5")
	assert.Contains(test, string(exiterr.Stderr), "first twin")
	assert.Contains(test, string(exiterr.Stderr), "second twin")

	_, exiterr, _ = program("cccc", "done")
	require.NotNil(test, exiterr)
	assert.Equal(test, 3, exiterr.ExitCode(), "a reference matching nothing is exit 3")

	_, exiterr, _ = program("bbb", "done")
	require.NotNil(test, exiterr)
	assert.Equal(test, 2, exiterr.ExitCode(), "a reference under four characters is exit 2")

	output, _, _ = program("show-open")
	assert.Len(
		test,
		unmarshalTaskArray(test, output),
		3,
		"no failed reference may resolve anything",
	)
}

// Issue #1: two clones each adding a task allocated the same numeric key, the
// merge was clean, and every command failed afterwards.
func TestDivergedClonesThatBothAddKeepWorking(test *testing.T) {
	first, cleanup := makeRepo(test)
	defer cleanup()

	output, exiterr, success := testCmd(first)("add", "shared")
	assertProgramResult(test, output, exiterr, success)

	second := filepath.Join(test.TempDir(), "second")
	runGit(test, "clone", "--quiet", first, second)

	output, exiterr, success = testCmd(first)("add", "from the first clone")
	assertProgramResult(test, output, exiterr, success)

	output, exiterr, success = testCmd(second)("add", "from the second clone")
	assertProgramResult(test, output, exiterr, success)

	runGit(test, "-C", first, "pull", "--quiet", "--no-rebase", "--no-edit", second, "HEAD")

	program := testCmd(first)

	output, exiterr, success = program("next")
	assertProgramResult(test, output, exiterr, success)
	assert.Len(test, unmarshalTaskArray(test, output), 3)

	for _, summary := range []string{"from the first clone", "from the second clone"} {
		output, exiterr, success = program(referenceFor(test, first, summary), "done")
		assertProgramResult(test, output, exiterr, success)
	}

	output, exiterr, success = program("show-resolved")
	assertProgramResult(test, output, exiterr, success)
	assert.Len(test, unmarshalTaskArray(test, output), 2)
}

const handWrittenRecord = `key: 4
summary: hand written
notes: |-
  first line 
  second line
priority: P2
created: 2026-01-01T00:00:00Z
`

// makeKeyedStore leaves a store the way the numeric-key versions did: a key in
// every record, resolved ones included, and a committed counter.
func makeKeyedStore(test *testing.T, repo string) []string {
	test.Helper()

	program := testCmd(repo)

	for _, summary := range []string{"open one", "open two", "finished"} {
		output, exiterr, success := program("add", summary)
		assertProgramResult(test, output, exiterr, success)
	}

	output, exiterr, success := program(referenceFor(test, repo, "finished"), "done")
	assertProgramResult(test, output, exiterr, success)

	records, err := filepath.Glob(filepath.Join(repo, "*", "*.yml"))
	require.NoError(test, err)
	require.Len(test, records, 3)

	for index, record := range records {
		stored, err := os.ReadFile(record)
		require.NoError(test, err)

		keyed := "key: " + string(rune('1'+index)) + "\n" + string(stored)
		require.NoError(test, os.WriteFile(record, []byte(keyed), 0o600))
	}

	// Styled by hand: the serialiser would quote these notes because of the
	// trailing space, so only a line-level edit leaves them as they are.
	handWritten := filepath.Join(repo, "pending", "cccc3333-0000-4000-8000-000000000004.yml")
	require.NoError(test, os.WriteFile(handWritten, []byte(handWrittenRecord), 0o600))
	records = append(records, handWritten)

	require.NoError(test, os.WriteFile(filepath.Join(repo, "next-key"), []byte("5\n"), 0o600))

	runGit(test, "-C", repo, "add", "-A")
	runGit(test, "-C", repo, "commit", "--quiet", "--no-gpg-sign", "-m", "keyed store")

	return records
}

func TestFirstLoadRemovesNumericKeysInOneCommit(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	records := makeKeyedStore(test, repo)
	before := runGit(test, "-C", repo, "rev-list", "--count", "HEAD")

	output, exiterr, success := testCmd(repo)("next")
	assertProgramResult(test, output, exiterr, success)
	assert.Len(test, unmarshalTaskArray(test, output), 3)

	for _, record := range records {
		stored, err := os.ReadFile(record)
		require.NoError(test, err)
		assert.NotContains(test, string(stored), "key:", "%s still has its key", record)
	}

	_, err := os.Stat(filepath.Join(repo, "next-key"))
	assert.True(test, os.IsNotExist(err), "the counter must be deleted")

	assert.Equal(
		test,
		"Remove numeric task keys",
		runGit(test, "-C", repo, "log", "-1", "--format=%s"),
	)

	afterMigration := runGit(test, "-C", repo, "rev-list", "--count", "HEAD")
	assert.Equal(
		test,
		increment(test, before),
		afterMigration,
		"the migration is exactly one commit",
	)

	// The commit removes the key lines and the counter, and nothing else.
	diff := runGit(test, "-C", repo, "show", "--format=", "--unified=0", "HEAD")
	changed := regexp.MustCompile(`(?m)^[-+][^-+].*$`).FindAllString(diff, -1)
	assert.ElementsMatch(test, []string{"-key: 1", "-key: 2", "-key: 3", "-key: 4", "-5"}, changed)

	handWritten, err := os.ReadFile(records[len(records)-1])
	require.NoError(test, err)
	assert.Equal(test, strings.TrimPrefix(handWrittenRecord, "key: 4\n"), string(handWritten))

	assert.Empty(test, runGit(test, "-C", repo, "status", "--porcelain", "--untracked-files=no"))

	output, exiterr, success = testCmd(repo)("show-resolved")
	assertProgramResult(test, output, exiterr, success)
	assert.Equal(test, afterMigration, runGit(test, "-C", repo, "rev-list", "--count", "HEAD"),
		"a second load of a migrated store changes nothing")
}

// The migration runs inside the load, so a commit it cannot make is reported
// and the command still runs.
func TestAMigrationThatCannotCommitDoesNotFailTheLoad(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	makeKeyedStore(test, repo)

	require.NoError(
		test,
		os.WriteFile(filepath.Join(repo, "unrelated.txt"), []byte("mine\n"), 0o600),
	)
	runGit(test, "-C", repo, "add", "unrelated.txt")

	output, exiterr, success := runBinary(repo, "next", "--json")
	assertProgramResult(test, output, exiterr, success)
	assert.Len(test, unmarshalTaskArray(test, output), 3)
	assert.Equal(test, "keyed store", runGit(test, "-C", repo, "log", "-1", "--format=%s"))
}

func increment(test *testing.T, count string) string {
	test.Helper()

	value, err := strconv.Atoi(count)
	require.NoError(test, err)

	return strconv.Itoa(value + 1)
}

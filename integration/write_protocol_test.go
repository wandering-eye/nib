package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// `git add .` staged the whole worktree, so anything a user left staged rode
// along inside a commit that claimed to be a task change. Refusing is louder
// than quietly excluding it: the user finds out their work is in the way.
func TestStagedWorkOutsideTaskPathsIsRefused(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	program := testCmd(repo)
	output, exiterr, success := program("add", "legitimate")
	assertProgramResult(test, output, exiterr, success)

	before, err := exec.Command("git", "-C", repo, "rev-list", "--count", "HEAD").Output()
	assert.NoError(test, err)

	stray := filepath.Join(repo, "unrelated.txt")
	assert.NoError(test, os.WriteFile(stray, []byte("not a task\n"), 0o600))
	assert.NoError(test, exec.Command("git", "-C", repo, "add", "unrelated.txt").Run())

	_, exiterr, _ = program("add", "second")
	assert.NotNil(test, exiterr, "a foreign staged file must stop the commit")

	after, err := exec.Command("git", "-C", repo, "rev-list", "--count", "HEAD").Output()
	assert.NoError(test, err)
	assert.Equal(test, strings.TrimSpace(string(before)), strings.TrimSpace(string(after)),
		"nothing may be committed while foreign work is staged")

	committed, err := exec.Command("git", "-C", repo, "log", "--name-only", "--format=").Output()
	assert.NoError(test, err)
	assert.NotContains(test, string(committed), "unrelated.txt")
}

// A status change empties the old directory, and that deletion has to be
// committed or the record exists under two statuses at once.
func TestAStatusChangeCommitsTheDeletion(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	program := testCmd(repo)
	output, exiterr, success := program("add", "moves status")
	assertProgramResult(test, output, exiterr, success)

	output, exiterr, success = program("start", referenceFor(test, repo, "moves status"))
	assertProgramResult(test, output, exiterr, success)

	tracked, err := exec.Command("git", "-C", repo, "ls-tree", "-r", "--name-only", "HEAD").Output()
	assert.NoError(test, err)

	var records int

	for _, line := range strings.Split(string(tracked), "\n") {
		if strings.HasSuffix(strings.TrimSpace(line), ".yml") {
			records++
		}
	}

	assert.Equal(test, 1, records, "the record must exist under exactly one status")

	status, err := exec.Command("git", "-C", repo, "status", "--porcelain").Output()
	assert.NoError(test, err)
	assert.NotContains(test, string(status), ".yml", "no record change may be left uncommitted")
}

// Racing creates must each land as their own record, and the store must still
// load afterwards.
func TestConcurrentCreatesAllLand(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	const creates = 20

	var waiting sync.WaitGroup

	for index := 0; index < creates; index++ {
		waiting.Add(1)

		go func(number int) {
			defer waiting.Done()

			_, _, _ = runBinary(repo, "add", "concurrent")
		}(index)
	}

	waiting.Wait()

	output, exiterr, success := testCmd(repo)("next")
	assertProgramResult(test, output, exiterr, success)

	tasks := unmarshalTaskArray(test, output)
	assert.Len(test, tasks, creates, "every create must have landed")

	seen := map[string]bool{}
	for _, task := range tasks {
		assert.False(test, seen[task.UUID], "%s was listed twice", task.UUID)
		seen[task.UUID] = true
	}
}

// A commit that did not move HEAD is not a commit, and an operation that
// changed nothing is not a success.
func TestRepeatedNoOpDoesNotCommit(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	program := testCmd(repo)
	output, exiterr, success := program("add", "already started")
	assertProgramResult(test, output, exiterr, success)

	reference := referenceFor(test, repo, "already started")

	output, exiterr, success = program("start", reference)
	assertProgramResult(test, output, exiterr, success)

	before, err := exec.Command("git", "-C", repo, "rev-list", "--count", "HEAD").Output()
	assert.NoError(test, err)

	_, exiterr, _ = program("start", reference)
	assert.NotNil(test, exiterr, "starting an active task changes nothing")
	assert.Equal(test, 4, exiterr.ExitCode(), "a no-op is exit 4")

	after, err := exec.Command("git", "-C", repo, "rev-list", "--count", "HEAD").Output()
	assert.NoError(test, err)
	assert.Equal(test, strings.TrimSpace(string(before)), strings.TrimSpace(string(after)),
		"a no-op must not create a commit")
}

// Removing the last task in a status directory stages a deletion that leaves
// the directory untracked, so the change has to be checked against the paths
// that were staged, not the ones left afterwards.
func TestRemovingTheLastTaskCommitsTheDeletion(test *testing.T) {
	repo, cleanup := makeRepo(test)
	defer cleanup()

	output, exiterr, success := testCmd(repo)("add", "only task")
	assertProgramResult(test, output, exiterr, success)

	output, exiterr, success = testConfirmingCmd(
		repo,
	)(
		"remove",
		referenceFor(test, repo, "only task"),
	)
	assertProgramResult(test, output, exiterr, success)

	tracked, err := exec.Command("git", "-C", repo, "ls-tree", "-r", "--name-only", "HEAD").Output()
	assert.NoError(test, err)
	assert.NotContains(test, string(tracked), "pending/")
}

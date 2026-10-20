package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The repository lock lives in the repository directory, so taking it before
// the create prompt failed on a fresh machine where that directory does not
// exist yet.
func TestFirstRunCreatesTheRepository(test *testing.T) {
	repo := filepath.Join(test.TempDir(), "store")

	output, exiterr, success := runBinaryWithInput(repo, "y\n", "add", "first task")
	assertProgramResult(test, output, exiterr, success)

	subjects, err := exec.Command("git", "-C", repo, "log", "--format=%s").Output()
	assert.NoError(test, err)
	assert.Contains(test, string(subjects), "first task")
}

func TestDecliningTheFirstRunCreatesNothing(test *testing.T) {
	repo := filepath.Join(test.TempDir(), "store")

	_, exiterr, _ := runBinary(repo, "add", "first task")
	assert.NotNil(test, exiterr, "end of input is a decline")

	_, err := os.Stat(repo)
	assert.True(test, os.IsNotExist(err), "a declined prompt must not leave a directory behind")
}

// Two first runs both see no repository and both get the prompt; the lock
// still serialises init and the commits, so both tasks land.
func TestConcurrentFirstRunsBothSucceed(test *testing.T) {
	repo := filepath.Join(test.TempDir(), "store")

	var group sync.WaitGroup

	failures := make(chan string, 2)

	for _, summary := range []string{"from one", "from two"} {
		group.Add(1)

		go func() {
			defer group.Done()

			output, exiterr, success := runBinaryWithInput(repo, "y\n", "add", summary)
			if !success {
				stderr := ""
				if exiterr != nil {
					stderr = string(exiterr.Stderr)
				}

				failures <- summary + ": " + string(output) + stderr
			}
		}()
	}

	group.Wait()
	close(failures)

	for failure := range failures {
		test.Error(failure)
	}

	output, exiterr, success := testCmd(repo)("next")
	assertProgramResult(test, output, exiterr, success)

	tasks := unmarshalTaskArray(test, output)
	assert.Len(test, tasks, 2)

	if len(tasks) == 2 {
		assert.NotEqual(test, tasks[0].UUID, tasks[1].UUID)
	}
}

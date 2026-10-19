package integration

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	dstask "github.com/wandering-eye/nib"
	"gotest.tools/assert"
)

func TestMain(m *testing.M) {
	binary := binaryPath()

	if err := compile(binary); err != nil {
		log.Fatalf("compile error: %v", err)
	}

	cleanup := func() {
		if err := os.Remove(binary); err != nil {
			log.Panicf("could not remove integration test binary %s", binary)
		}
	}

	exitCode := m.Run()

	cleanup()
	os.Exit(exitCode)
}

func compile(outputPath string) error {
	// We expect to execute in the ./integration directory, and we will output
	// our test binary there.
	cmd := exec.Command("go", "build", "-o", outputPath, "../cmd/nib/main.go")

	return cmd.Run()
}

func binaryPath() string {
	if runtime.GOOS == "windows" {
		return "./nib.exe"
	}

	return "./nib"
}

// Create a callable closure that will run our test binary against a
// particular repository path. Any variables set in the environment will be
func testCmd(repoPath string) func(args ...string) ([]byte, *exec.ExitError, bool) {
	return func(args ...string) ([]byte, *exec.ExitError, bool) {
		// These tests assert on task fields, so they ask for the machine
		// format. Output no longer changes shape when piped, so a test that
		// wants JSON has to say so.
		return runBinary(repoPath, append(args, "--json")...)
	}
}

// testTableCmd runs the binary without --json, so the default human table is
// exercised rather than only the machine format.
func testTableCmd(repoPath string) func(args ...string) ([]byte, *exec.ExitError, bool) {
	return func(args ...string) ([]byte, *exec.ExitError, bool) {
		return runBinary(repoPath, args...)
	}
}

// testConfirmingCmd answers the confirmation prompt. A bulk change with no IDs
// asks before touching every task in the context, and end of input counts as a
// decline, so a test that means to go ahead has to say so.
func testConfirmingCmd(repoPath string) func(args ...string) ([]byte, *exec.ExitError, bool) {
	return func(args ...string) ([]byte, *exec.ExitError, bool) {
		return runBinaryWithInput(repoPath, "y\n", append(args, "--json")...)
	}
}

func runBinary(repoPath string, args ...string) ([]byte, *exec.ExitError, bool) {
	return runBinaryWithInput(repoPath, "", args...)
}

func runBinaryWithInput(repoPath, input string, args ...string) ([]byte, *exec.ExitError, bool) {
	cmd := exec.Command(binaryPath(), args...)
	cmd.Env = repoEnvironment(repoPath)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	output, err := cmd.Output()
	exitErr := &exec.ExitError{}
	ok := errors.As(err, &exitErr)

	if !ok {
		if err != nil {
			return output, nil, false
		}

		return output, nil, true
	}

	return output, exitErr, exitErr.Success()
}

// repoEnvironment points both repository variables at the test repository, so
// neither one exported in the developer's shell can reach their real store.
// NIB_GIT_REPO wins over DSTASK_GIT_REPO, and the last duplicate entry wins.
func repoEnvironment(repoPath string) []string {
	return append(os.Environ(), "NIB_GIT_REPO="+repoPath, "DSTASK_GIT_REPO="+repoPath)
}

// Sets an environment variable, and returns a callable closure to unset it.
func setEnv(key, value string) func() {
	if err := os.Setenv(key, value); err != nil {
		panic(err)
	}

	return func() {
		if err := os.Unsetenv(key); err != nil {
			panic(err)
		}
	}
}

func logFailure(t *testing.T, output []byte, exiterr *exec.ExitError) {
	t.Helper()
	t.Logf("stdout: %s", string(output))
	if exiterr != nil {
		t.Logf("stderr: %s", string(exiterr.Stderr))
	} else {
		t.Log("stderr: <nil>")
	}
}

func unmarshalTaskArray(t *testing.T, data []byte) []dstask.Task {
	t.Helper()

	var tasks []dstask.Task
	err := json.Unmarshal(data, &tasks)
	assert.NilError(t, err)

	return tasks
}

func unmarshalProjectArray(t *testing.T, data []byte) []dstask.Project {
	t.Helper()

	var projects []dstask.Project
	err := json.Unmarshal(data, &projects)
	assert.NilError(t, err)

	return projects
}

func makeDstaskRepo(t *testing.T) (string, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "dstask")
	if err != nil {
		t.Fatal()
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = dir

	if err := cmd.Run(); err != nil {
		t.Fatal()
	}

	cleanup := func() {
		if err := os.RemoveAll(dir); err != nil {
			log.Printf("failed to remove temporary directory %s: %v", dir, err)
		}
	}

	return dir, cleanup
}

func assertProgramResult(
	t *testing.T,
	output []byte,
	exiterr *exec.ExitError,
	successExpected bool,
) {
	t.Helper()

	if exiterr != nil || !successExpected {
		logFailure(t, output, exiterr)
		if exiterr != nil {
			t.Fatalf("%v", exiterr)
		}
		t.Fatalf("command exited unsuccessfully")
	}
}

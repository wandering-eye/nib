package dstask

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// RunGitCmd shells out to git in the context of the nib repository.
// RunGitCmdWithOutput captures stdout rather than inheriting it, for the few
// places that need to read git's answer instead of showing it.
func RunGitCmdWithOutput(repoPath string, args ...string) (string, error) {
	args = append([]string{"-C", repoPath}, args...)

	out, err := exec.Command("git", args...).Output()

	return string(out), err
}

func RunGitCmd(repoPath string, args ...string) error {
	args = append([]string{"-C", repoPath}, args...)

	return RunCmd("git", args...)
}

// MustRunGitCmd delegates to RunGitCmd and exits the program on any error.
func MustRunGitCmd(repoPath string, args ...string) {
	err := RunGitCmd(repoPath, args...)
	if err != nil {
		ExitFail("Failed to run git cmd.")
	}
}

// MustGitCommit is like GitCommit, except if any error is
// encountered, the program exits.
func MustGitCommit(repoPath, format string, a ...any) {
	msg := fmt.Sprintf(format, a...)

	fmt.Printf("\n%s\n", msg)
	fmt.Printf("\033[38;5;245m")

	if err := GitCommit(repoPath, format, a...); err != nil {
		// Preserve the code: a commit that changed nothing is a no-op, not a
		// failure, and callers distinguish them.
		ExitError(err)
	}

	fmt.Printf("\033[0m")
}

// GitCommit stages changes in the dstask repository and commits them.
// gitPaths are the only paths a commit is allowed to touch: the status
// directories and the key high water mark. `git add .` used to stage the whole
// worktree, so anything else a user happened to leave staged rode along in a
// commit describing a task change. Paths that do not exist are left out,
// because git rejects a pathspec that matches nothing.
func gitPaths(repoPath string) []string {
	candidates := append(append([]string{}, ALL_STATUSES...), "next-key")

	paths := make([]string, 0, len(candidates))

	for _, candidate := range candidates {
		// An empty status directory is skipped: git tracks no such path and
		// rejects a pathspec that matches nothing.
		if !hasContent(repoPath, candidate) {
			continue
		}

		paths = append(paths, candidate)
	}

	return paths
}

// hasContent reports whether git needs to see this path: either something is
// there now, or git is tracking something that used to be. The second case
// matters because a status change empties the old directory, and the deletion
// still has to be staged.
func hasContent(repoPath, candidate string) bool {
	full := filepath.Join(repoPath, candidate)

	if info, err := os.Stat(full); err == nil {
		if !info.IsDir() {
			return true
		}

		if entries, err := os.ReadDir(full); err == nil && len(entries) > 0 {
			return true
		}
	}

	tracked, err := RunGitCmdWithOutput(repoPath, "ls-files", "--", candidate)

	return err == nil && strings.TrimSpace(tracked) != ""
}

func GitCommit(repoPath, format string, a ...any) error {
	msg := fmt.Sprintf(format, a...)

	head, hadHead := gitHead(repoPath)

	// Anything already staged would be swept into a commit describing a task
	// change, which is how `git add .` used to behave. Refuse instead of
	// quietly committing someone else's work under our message.
	foreign, err := stagedOutsideTaskPaths(repoPath)
	if err != nil {
		return err
	}

	if len(foreign) > 0 {
		return fmt.Errorf(
			"refusing to commit: %s is staged and is not task state; commit or unstage it first",
			strings.Join(foreign, ", "),
		)
	}

	arguments := append([]string{"add", "--"}, gitPaths(repoPath)...)
	if _, err := RunGitCmdWithOutput(repoPath, arguments...); err != nil {
		return fmt.Errorf("failed to add changes to repo: %w", err)
	}

	// Scoped to the task paths: an unstaged edit to some other tracked file
	// would otherwise look like a change, skip the no-op result, and turn a
	// command that did nothing into a commit failure.
	noChange := append([]string{"diff", "--cached", "--quiet", "HEAD", "--"}, gitPaths(repoPath)...)
	if _, noChangeErr := RunGitCmdWithOutput(repoPath, noChange...); hadHead && noChangeErr == nil {
		return NoOpError("nothing changed")
	}

	if _, err := RunGitCmdWithOutput(repoPath, "commit", "--no-gpg-sign", "-m", msg); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	// The write is only done when history says so. Reporting success off an
	// exit code alone is how a change that never landed looked like one that
	// did.
	moved, hasHead := gitHead(repoPath)
	if !hasHead || (hadHead && moved == head) {
		return fmt.Errorf("commit reported success but HEAD did not move")
	}

	return nil
}

// stagedOutsideTaskPaths lists staged files that are not task state.
func stagedOutsideTaskPaths(repoPath string) ([]string, error) {
	out, err := RunGitCmdWithOutput(repoPath, "diff", "--cached", "--name-only")
	if err != nil {
		return nil, fmt.Errorf("failed to inspect staged changes: %w", err)
	}

	owned := map[string]bool{}
	for _, path := range append(append([]string{}, ALL_STATUSES...), "next-key") {
		owned[path] = true
	}

	var foreign []string

	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if !owned[strings.SplitN(trimmed, "/", 2)[0]] {
			foreign = append(foreign, trimmed)
		}
	}

	return foreign, nil
}

// gitHead returns the current commit, and whether there is one at all.
func gitHead(repoPath string) (string, bool) {
	out, err := RunGitCmdWithOutput(repoPath, "rev-parse", "HEAD")
	if err != nil {
		return "", false
	}

	return strings.TrimSpace(out), true
}

// MustGetRepoPath returns the full path to a file within the dstask git repo.
// Pass file as an empty string to return the git repo directory itself.
func MustGetRepoPath(repoPath, directory, file string) string {
	dir := path.Join(repoPath, directory)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0o700)
		if err != nil {
			ExitFail("Failed to create directory in git repository")
		}
	}

	return path.Join(dir, file)
}

// EnsureRepoExists checks for the existence of a dstask repository, or exits the program.
func EnsureRepoExists(repoPath string) {
	_, err := exec.LookPath("git")
	if err != nil {
		ExitFail("git required, please install")
	}

	gitDotGitLocation := path.Join(repoPath, ".git")
	if _, err := os.Stat(gitDotGitLocation); os.IsNotExist(err) {
		ConfirmOrAbort("could not find a nib repository at %s -- create?", repoPath)

		err = os.Mkdir(repoPath, 0o700)
		if err != nil {
			ExitFail("Failed to create directory in git repository")
		}

		MustRunGitCmd(repoPath, "init")
		fmt.Println("\nAdd a remote repository with:\n\n\tdstask git remote add origin <repo>")
		fmt.Println() // must be a separate call else compiler complains of redundant \n
	}
}

// Sync performs a git pull, and then a git push. If any conflicts are encountered,
// the user will need to resolve them.
func Sync(repoPath string) {
	MustRunGitCmd(repoPath, "pull", "--ff", "--no-rebase", "--no-edit", "--commit")
	MustRunGitCmd(repoPath, "push")
}

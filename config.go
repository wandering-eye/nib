package dstask

import (
	"os"
	"path/filepath"
)

// Config models the dstask application's required configuration. All paths
// are absolute.
type Config struct {
	// Path to the git repository
	Repo string
	// Path to the dstask local state file. State will differ between machines
	StateFile string
	// Path to the ids file
	// An unparsed context string, provided via DSTASK_CONTEXT
	CtxFromEnvVar string
}

// NewConfig generates a new Config struct from the environment.
func NewConfig() Config {
	var conf Config

	conf.CtxFromEnvVar = getEnv("DSTASK_CONTEXT", "")
	// Determine home directory in a platform-independent way
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: use $HOME if present
		home = os.Getenv("HOME")
	}
	// The store keeps its existing location. nib reads a dstask repository as
	// it stands, so moving the default would strand the data rather than
	// upgrade it. DSTASK_GIT_REPO still works for anyone who already set it.
	defaultRepo := filepath.Join(home, ".dstask")
	conf.Repo = getEnv("NIB_GIT_REPO", getEnv("DSTASK_GIT_REPO", defaultRepo))
	conf.StateFile = filepath.Join(conf.Repo, ".git", "dstask", "state.bin")

	return conf
}

// getEnv returns an env var's value, or a default.
func getEnv(key string, _default string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return _default
}

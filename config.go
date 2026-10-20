package nib

import (
	"os"
	"path/filepath"
)

// Config models the nib application's required configuration. All paths
// are absolute.
type Config struct {
	// Path to the git repository
	Repo string
	// Path to the nib local state file. State will differ between machines
	StateFile string
	// Path to the ids file
	// An unparsed context string, provided via NIB_CONTEXT
	CtxFromEnvVar string
}

// NewConfig generates a new Config struct from the environment.
func NewConfig() Config {
	var conf Config

	conf.CtxFromEnvVar = getEnv("NIB_CONTEXT", "")
	// Determine home directory in a platform-independent way
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: use $HOME if present
		home = os.Getenv("HOME")
	}
	// There is no fallback to an older location or variable: a store kept
	// elsewhere is moved to ~/.nib or named with NIB_GIT_REPO.
	defaultRepo := filepath.Join(home, ".nib")
	conf.Repo = getEnv("NIB_GIT_REPO", defaultRepo)
	conf.StateFile = filepath.Join(conf.Repo, ".git", "nib", "state.bin")

	return conf
}

// getEnv returns an env var's value, or a default.
func getEnv(key string, _default string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return _default
}

package nib

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultStoreIsInHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("NIB_GIT_REPO", "")

	conf := NewConfig()

	assert.Equal(t, filepath.Join(home, ".nib"), conf.Repo)
	assert.Equal(t, filepath.Join(home, ".nib", ".git", "nib", "state.bin"), conf.StateFile)
}

func TestRepoVariableOverridesTheDefault(t *testing.T) {
	repo := t.TempDir()
	t.Setenv("NIB_GIT_REPO", repo)

	assert.Equal(t, repo, NewConfig().Repo)
}

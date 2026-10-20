package nib

import (
	"encoding/gob"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The saved context is a gob of Query, which used to carry numeric IDs and a
// numeric template key. Those fields no longer exist under those names, so an
// old state file must still load, keeping everything else and dropping them.
func TestStateSavedWithNumericKeysStillLoads(t *testing.T) {
	type legacyQuery struct {
		Cmd      string
		IDs      []int
		Tags     []string
		Project  string
		Template int
	}

	type legacyState struct {
		Context legacyQuery
	}

	path := filepath.Join(t.TempDir(), "state.bin")

	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, gob.NewEncoder(file).Encode(legacyState{
		Context: legacyQuery{IDs: []int{3}, Tags: []string{"work"}, Project: "nib", Template: 7},
	}))
	require.NoError(t, file.Close())

	state := LoadState(path)

	assert.Equal(t, []string{"work"}, state.Context.Tags)
	assert.Equal(t, "nib", state.Context.Project)
	assert.Empty(t, state.Context.References)
	assert.Empty(t, state.Context.TemplateReference)
}

package nib

// this file represents the interface to the state specific to the PC nib is
// on is stored. This is very minimal at the moment -- just the current
// context. It will probably remain that way.

import (
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
)

// State models our local context for serialisation and deserialisation from
// our state file.
type State struct {
	// Context is an implicit command line that changes the behavior or display
	// of some commands.
	Context Query
}

// Save serialises State to disk as gob binary data.
func (state State) Save(stateFilePath string) {
	if err := os.MkdirAll(filepath.Dir(stateFilePath), os.ModePerm); err != nil {
		ExitFail("Failed to create directories for %s: %s", stateFilePath, err)
	}

	mustWriteGob(stateFilePath, &state)
}

// LoadState reads the state file, if it exists. Otherwise a default State is returned.
func LoadState(stateFilePath string) State {
	if _, err := os.Stat(stateFilePath); os.IsNotExist(err) {
		return State{}
	}

	state := State{}
	mustReadGob(stateFilePath, &state)

	return state
}

// SetContext sets a context on State, with some validation.
func (state *State) SetContext(context Query) error {
	if len(context.IDs) != 0 {
		return errors.New("context cannot contain IDs")
	}

	if context.Text != "" {
		return errors.New("context cannot contain text")
	}

	state.Context = context

	return nil
}

func mustWriteGob(filePath string, object any) {
	file, err := os.Create(filePath)
	if err != nil {
		ExitFail("Failed to open %s for writing: ", filePath)
	}

	defer func() {
		if err := file.Close(); err != nil {
			ExitFail("Failed to close file: %v", err)
		}
	}()

	encoder := gob.NewEncoder(file)

	err = encoder.Encode(object)
	if err != nil {
		ExitFail("Failed to encode state gob: %s, %s", filePath, err)
	}
}

func mustReadGob(filePath string, object any) {
	file, err := os.Open(filePath)
	if err != nil {
		ExitFail("Failed to open %s for reading: ", filePath)
	}

	defer func() {
		if err := file.Close(); err != nil {
			ExitFail("Failed to close file: %v", err)
		}
	}()

	decoder := gob.NewDecoder(file)

	err = decoder.Decode(object)
	if err != nil {
		ExitFail("Failed to parse state gob: %s, %s", filePath, err)
	}
}

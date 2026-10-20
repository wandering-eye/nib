package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/wandering-eye/nib"
	"github.com/wandering-eye/nib/completions"
)

func main() {
	query := nib.ParseQuery(os.Args[1:]...)

	// It will remain true if we handle a command that doesn't require
	// initialisation
	noInitCommand := true

	// Handle commands that don't require initialisation
	switch query.Cmd {
	case nib.CMD_HELP:
		nib.CommandHelp(os.Args)

	case nib.CMD_VERSION:
		nib.CommandVersion()

	case nib.CMD_PRINT_BASH_COMPLETION:
		fmt.Print(completions.Bash)

	case nib.CMD_PRINT_ZSH_COMPLETION:
		fmt.Print(completions.Zsh)

	case nib.CMD_PRINT_FISH_COMPLETION:
		fmt.Print(completions.Fish)

	case nib.CMD_PRINT_NU_COMPLETION:
		fmt.Print(completions.Nushell)

	default:
		noInitCommand = false
	}

	if noInitCommand {
		return
	}

	conf := nib.NewConfig()

	// The lock file needs the directory, so the create prompt comes first.
	nib.PrepareRepoDirectory(conf.Repo)

	// Taken before the repository is initialised: two first invocations would
	// otherwise race to init it. Held for the whole command so a read, the
	// write and the commit that follows are one step. The kernel holds it, so a
	// process exiting releases it.
	unlock, err := nib.LockRepo(conf.Repo)
	if err != nil {
		nib.ExitError(err)
	}

	defer unlock()

	nib.InitRepoIfMissing(conf.Repo)

	// Load state for getting and setting ctx
	state := nib.LoadState(conf.StateFile)
	ctx := state.Context

	// Check if we have a context override.
	if conf.CtxFromEnvVar != "" {
		if query.Cmd == nib.CMD_CONTEXT && len(os.Args) >= 3 {
			nib.ExitFail("setting context not allowed while NIB_CONTEXT is set")
		}

		splitted := strings.Fields(conf.CtxFromEnvVar)
		ctx = nib.ParseQuery(splitted...)
	}

	// Check if we ignore context with the "--" token
	if query.IgnoreContext {
		ctx = nib.Query{}
	}

	switch query.Cmd {
	// The default command
	case "", nib.CMD_NEXT, nib.CMD_SHOW_NEXT:
		if err := nib.CommandNext(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_SHOW_OPEN:
		if err := nib.CommandShowOpen(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_ADD:
		if err := nib.CommandAdd(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_RM, nib.CMD_REMOVE:
		if err := nib.CommandRemove(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_TEMPLATE:
		if err := nib.CommandTemplate(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_LOG:
		if err := nib.CommandLog(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_START:
		if err := nib.CommandStart(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_STOP:
		if err := nib.CommandStop(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_DONE, nib.CMD_RESOLVE:
		if err := nib.CommandDone(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_CONTEXT:
		if err := nib.CommandContext(conf, state, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_MODIFY:
		if err := nib.CommandModify(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_EDIT:
		if err := nib.CommandEdit(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_NOTE, nib.CMD_NOTES:
		if err := nib.CommandNote(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_UNDO:
		if err := nib.CommandUndo(conf, os.Args, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_SYNC:
		if err := nib.CommandSync(conf.Repo); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_GIT:
		nib.MustRunGitCmd(conf.Repo, os.Args[2:]...)

	case nib.CMD_SHOW_ACTIVE:
		if err := nib.CommandShowActive(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_SHOW_PAUSED:
		if err := nib.CommandShowPaused(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_OPEN:
		if err := nib.CommandOpen(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_SHOW_PROJECTS:
		if err := nib.CommandShowProjects(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_SHOW_TAGS:
		if err := nib.CommandShowTags(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_SHOW_TEMPLATES:
		if err := nib.CommandShowTemplates(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_SHOW_RESOLVED:
		if err := nib.CommandShowResolved(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_SHOW_UNORGANISED:
		if err := nib.CommandShowUnorganised(conf, ctx, query); err != nil {
			nib.ExitError(err)
		}

	case nib.CMD_COMPLETIONS:
		completions.Completions(conf, os.Args, ctx)

	default:
		panic("this should never happen?")
	}
}

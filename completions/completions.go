package completions

import (
	"fmt"
	"log"
	"strings"

	"github.com/wandering-eye/nib"
)

// Completions ...
func Completions(conf nib.Config, args []string, ctx nib.Query) {
	// given the entire user's command line arguments as the arguments for
	// this cmd, suggest possible candidates for the last arg.
	// see the relevant shell completion bindings in this repository for
	// integration. Note there are various idiosyncrasies with bash
	// involving arg separation.
	var completions []string

	var originalArgs []string

	var prefix string

	// drop nib _completions nib to allow parsing what is on actual
	// prompt
	if len(args) > 3 {
		originalArgs = args[3:]
	}

	// args are nib _completions <user command line>
	// parse command line as normal to set rules
	query := nib.ParseQuery(originalArgs...)

	// No command and OK to specify command (to run or help)
	// Note that techically we should only specify commands as available
	// completions if the last partial argument is a command substring.
	// However, this is unnecessary as a general substring filter is used at
	// the end of the func.
	// This is exhaustive but the clearest way, IMO.
	if len(query.AntiProjects) == 0 &&
		query.Project == "" &&
		len(query.Tags) == 0 &&
		len(query.AntiTags) == 0 &&
		query.Priority == "" &&
		query.TemplateReference == "" &&
		!query.IgnoreContext &&
		(query.Cmd == nib.CMD_HELP || query.Cmd == "") {
		for _, cmd := range nib.ALL_CMDS {
			if !strings.HasPrefix(cmd, "_") {
				completions = append(completions, cmd)
			}
		}
	}

	if nib.StrSliceContains([]string{
		"",
		nib.CMD_NEXT,
		nib.CMD_ADD,
		nib.CMD_REMOVE,
		nib.CMD_LOG,
		nib.CMD_START,
		nib.CMD_STOP,
		nib.CMD_DONE,
		nib.CMD_RESOLVE,
		nib.CMD_CONTEXT,
		nib.CMD_MODIFY,
		nib.CMD_SHOW_NEXT,
		nib.CMD_SHOW_PROJECTS,
		nib.CMD_SHOW_ACTIVE,
		nib.CMD_SHOW_PAUSED,
		nib.CMD_SHOW_OPEN,
		nib.CMD_SHOW_RESOLVED,
		nib.CMD_SHOW_TEMPLATES,
	}, query.Cmd) {
		ts, err := nib.LoadTaskSet(conf.Repo, false)
		if err != nil {
			log.Printf("completions error: %v\n", err)

			return
		}
		// limit completions to available context, but not if the user is
		// trying to change context, context ignore is on, or modify
		// command is being completed
		if !query.IgnoreContext &&
			query.Cmd != nib.CMD_CONTEXT &&
			query.Cmd != nib.CMD_MODIFY {
			if err := ts.Filter(ctx); err != nil {
				log.Printf("completions error: %v\n", err)

				return
			}
		}

		// templates
		if query.Cmd == nib.CMD_ADD {
			for _, task := range ts.Tasks() {
				if task.Status == nib.STATUS_TEMPLATE {
					completions = append(completions, "template:"+task.ShortUUID())
				}
			}
		}

		// priorities
		completions = append(completions, nib.PRIORITY_CRITICAL)
		completions = append(completions, nib.PRIORITY_HIGH)
		completions = append(completions, nib.PRIORITY_NORMAL)
		completions = append(completions, nib.PRIORITY_LOW)

		// projects
		for _, project := range ts.GetProjects() {
			completions = append(completions, "project:"+project.Name)
			completions = append(completions, "-project:"+project.Name)
		}

		// tags
		for tag := range ts.GetTags() {
			completions = append(completions, "+"+tag)
			completions = append(completions, "-"+tag)
		}
	}

	if len(originalArgs) > 0 {
		prefix = originalArgs[len(originalArgs)-1]
	}

	for _, completion := range completions {
		if strings.HasPrefix(completion, prefix) &&
			!nib.StrSliceContains(originalArgs, completion) {
			fmt.Println(completion)
		}
	}
}

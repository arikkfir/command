package command

import (
	"context"
	"errors"
	"fmt"
	"io"
)

type ExitCode int

const (
	ExitCodeSuccess          ExitCode = 0
	ExitCodeError            ExitCode = 1
	ExitCodeMisconfiguration ExitCode = 2
)

// Execute the correct command in the given command hierarchy (starting at "root"), configured from the given CLI args
// and environment variables. The command will be executed with the given context after all pre-RunFunc hooks have been
// successfully executed in the command hierarchy.
func Execute(ctx context.Context, w io.Writer, root *Command, args []string, envVars map[string]string) ExitCode {
	if root.parent != nil {
		_, _ = fmt.Fprintf(w, "%s: command must be the root command", errors.ErrUnsupported)
		return ExitCodeError
	}

	// Extract the command, CLI flags, positional arguments & the command hierarchy
	flags, positionals, cmd := root.inferCommandAndArgs(args)

	// Create flagSet & apply it to the configuration structs
	// If "--help" is given, print help and exit
	if err := cmd.flags.apply(envVars, append(flags, positionals...)); err != nil {
		_, _ = fmt.Fprintln(w, err)
		if err := cmd.PrintUsageLine(w, getTerminalWidth()); err != nil {
			_, _ = fmt.Fprintf(w, "%s\n", err)
			return ExitCodeError
		} else {
			return ExitCodeMisconfiguration
		}
	} else if cmd.HelpConfig.Help {
		if err := cmd.PrintHelp(w, getTerminalWidth()); err != nil {
			_, _ = fmt.Fprintf(w, "%s\n", err)
			return ExitCodeError
		} else {
			return ExitCodeSuccess
		}
	}

	// Invoke all "PreRun" hooks on the whole chain of commands (starting at the root)
	for _, c := range cmd.getChain() {
		for _, hook := range c.preRunHooks {
			if err := hook.PreRun(ctx); err != nil {
				_, _ = fmt.Fprintln(w, err)
				return ExitCodeError
			}
		}
	}

	// Run the command or print help screen if it's not a command
	if cmd.action != nil {
		if err := cmd.action.Run(ctx); err != nil {
			_, _ = fmt.Fprintln(w, err)
			return ExitCodeError
		}
	} else {
		// Command is not a runner - print help
		if err := cmd.PrintHelp(w, getTerminalWidth()); err != nil {
			_, _ = fmt.Fprintf(w, "%s\n", err)
			return ExitCodeError
		}
	}
	return ExitCodeSuccess

}

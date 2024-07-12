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

// ExecuteWithContext the correct command in the given command hierarchy (starting at "root"), configured from the given
// CLI args and environment variables. The command will be executed with the given context after all pre-RunFunc hooks
// have been successfully executed in the command hierarchy.
func ExecuteWithContext(ctx context.Context, w io.Writer, root *Command, args []string, envVars map[string]string) (exitCode ExitCode) {
	exitCode = ExitCodeSuccess

	// We insist on getting the root command - so that we can infer correctly which command the user wanted to invoke
	if root.parent != nil {
		_, _ = fmt.Fprintf(w, "%s: command must be the root command", errors.ErrUnsupported)
		exitCode = ExitCodeError
		return
	}

	// Extract the command, CLI flags, positional arguments & the command hierarchy
	flags, positionals, cmd := root.inferCommandAndArgs(args)

	// Create flagSet & apply it to the configuration structs
	// If "--help" is given, print help and exit
	if err := cmd.flags.apply(envVars, append(flags, positionals...)); err != nil {
		_, _ = fmt.Fprintln(w, err)
		if err := cmd.PrintUsageLine(w, getTerminalWidth()); err != nil {
			_, _ = fmt.Fprintf(w, "%s\n", err)
			exitCode = ExitCodeError
			return
		} else {
			exitCode = ExitCodeMisconfiguration
			return
		}
	} else if cmd.HelpConfig.Help {
		if err := cmd.PrintHelp(w, getTerminalWidth()); err != nil {
			_, _ = fmt.Fprintf(w, "%s\n", err)
			exitCode = ExitCodeMisconfiguration
			return
		} else {
			exitCode = ExitCodeSuccess
			return
		}
	}

	// Results
	var actionError error

	// Ensure we invoke post-run hooks before we return
	chain := cmd.getChain()
	defer func() {
		for i := len(chain) - 1; i >= 0; i-- {
			c := chain[i]
			for j := len(c.postRunHooks) - 1; j >= 0; j-- {
				h := c.postRunHooks[j]
				if err := h.PostRun(ctx, actionError, exitCode); err != nil {
					_, _ = fmt.Fprintln(w, err)
					exitCode = ExitCodeError
				}
			}
		}
	}()

	// Invoke all "PreRun" hooks on the whole chain of commands (starting at the root)
	for i := 0; i < len(chain); i++ {
		c := chain[i]
		for j := 0; j < len(c.preRunHooks); j++ {
			h := c.preRunHooks[j]
			if err := h.PreRun(ctx); err != nil {
				_, _ = fmt.Fprintln(w, err)
				actionError = err
				exitCode = ExitCodeError
				return
			}
		}
	}

	// Run the command or print help screen if it's not a command
	if cmd.action != nil {
		if err := cmd.action.Run(ctx); err != nil {
			_, _ = fmt.Fprintln(w, err)
			actionError = err
			exitCode = ExitCodeError
		}
	} else {
		// Command is not a runner - print help
		if err := cmd.PrintHelp(w, getTerminalWidth()); err != nil {
			_, _ = fmt.Fprintf(w, "%s\n", err)
			actionError = err
			exitCode = ExitCodeError
		}
	}
	return
}

// Execute the correct command in the given command hierarchy (starting at "root"), configured from the given
// CLI args and environment variables. The command will be executed with a context that gets canceled when an OS signal
// for termination is received, after all pre-RunFunc hooks have been successfully executed in the command hierarchy.
//
//goland:noinspection GoUnusedExportedFunction
func Execute(w io.Writer, root *Command, args []string, envVars map[string]string) ExitCode {
	// Prepare a context that gets canceled if OS termination signals are sent
	ctx, cancel := context.WithCancel(SetupSignalHandler())
	defer cancel()

	return ExecuteWithContext(ctx, w, root, args, envVars)
}

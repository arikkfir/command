package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
)

var (
	ErrInvalidCommand          = errors.New("invalid command")
	ErrCommandAlreadyHasParent = errors.New("command already has a parent")
)

// HelpConfig is a configuration added to every executed command, for automatic help screen generation.
type HelpConfig struct {
	Help bool `inherited:"true" desc:"Show this help screen and exit."`
}

type Action interface {
	Run(context.Context) error
}

type ActionFunc func(context.Context) error

func (i ActionFunc) Run(ctx context.Context) error {
	if i != nil {
		return i(ctx)
	} else {
		return nil
	}
}

type PreRunHook interface {
	PreRun(context.Context) error
}

type PreRunHookFunc func(context.Context) error

func (i PreRunHookFunc) PreRun(ctx context.Context) error {
	if i != nil {
		return i(ctx)
	} else {
		return nil
	}
}

type PostRunHook interface {
	PostRun(context.Context, error, ExitCode) error
}

type PostRunHookFunc func(context.Context, error, ExitCode) error

func (i PostRunHookFunc) PostRun(ctx context.Context, err error, exitCode ExitCode) error {
	if i != nil {
		return i(ctx, err, exitCode)
	} else {
		return nil
	}
}

// Command is a command instance, created by [New] and can be composed with more Command instances to form a CLI command
// hierarchy.
type Command struct {
	name             string
	shortDescription string
	longDescription  string
	preRunHooks      []PreRunHook
	postRunHooks     []PostRunHook
	action           Action
	flags            *flagSet
	parent           *Command
	subCommands      []*Command
	HelpConfig       *HelpConfig
}

// MustNew creates a new command using [New], but will panic if it returns an error.
//
//goland:noinspection GoUnusedExportedFunction
func MustNew(name, shortDescription, longDescription string, action Action, hooks []any, subCommands ...*Command) *Command {
	cmd, err := New(name, shortDescription, longDescription, action, hooks, subCommands...)
	if err != nil {
		panic(err)
	}
	return cmd
}

// New creates a new command with the given name, short & long descriptions, and the given executor. The executor object
// is also scanned for configuration structs via reflection.
func New(name, shortDescription, longDescription string, action Action, hooks []any, subCommands ...*Command) (*Command, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: empty name", ErrInvalidCommand)
	} else if shortDescription == "" {
		return nil, fmt.Errorf("%w: empty short description", ErrInvalidCommand)
	}

	// Translate the any-based hooks list into pre-run and post-run hooks
	// Fail on any hook that doesn't implement at least one of them
	var preRunHooks []PreRunHook
	var postRunHooks []PostRunHook
	for i, hook := range hooks {
		var pre, post bool
		if preRunHook, ok := hook.(PreRunHook); ok {
			preRunHooks = append(preRunHooks, preRunHook)
			pre = true
		}
		if postRunHook, ok := hook.(PostRunHook); ok {
			postRunHooks = append(postRunHooks, postRunHook)
			post = true
		}
		if !pre && !post {
			return nil, fmt.Errorf("%w: hook %d (%T) is neither a PreRunHook nor a PostRunHook", ErrInvalidCommand, i, hook)
		}
	}

	// Create the command instance
	cmd := &Command{
		name:             name,
		shortDescription: shortDescription,
		longDescription:  longDescription,
		action:           action,
		preRunHooks:      preRunHooks,
		postRunHooks:     postRunHooks,
		HelpConfig:       &HelpConfig{},
	}

	// Set nil parent
	if err := cmd.setParent(nil); err != nil {
		return nil, fmt.Errorf("failed creating command '%s': %w", name, err)
	}

	// Add sub-commands
	for _, subCmd := range subCommands {
		if err := cmd.AddSubCommand(subCmd); err != nil {
			return nil, fmt.Errorf("%w: failed adding sub-command '%s' to '%s': %w", ErrInvalidCommand, subCmd.name, name, err)
		}
	}

	return cmd, nil
}

// setParent updates the parent command of this command.
func (c *Command) setParent(parent *Command) error {

	// Determine the parent flagSet, if any
	var parentFlags *flagSet
	if parent != nil {
		parentFlags = parent.flags
	} else if parentFlagSet, err := newFlagSet(nil, reflect.ValueOf(c).Elem().FieldByName("HelpConfig")); err != nil {
		return fmt.Errorf("failed creating Help flag set: %w", err)
	} else {
		parentFlags = parentFlagSet
	}

	// Create the flag-set
	var configObjects []reflect.Value
	if c.action != nil {
		configObjects = append(configObjects, reflect.ValueOf(c.action))
	}
	for _, hook := range c.preRunHooks {
		configObjects = append(configObjects, reflect.ValueOf(hook))
	}
	if fs, err := newFlagSet(parentFlags, configObjects...); err != nil {
		return fmt.Errorf("failed creating flag-set for command '%s': %w", c.name, err)
	} else {
		c.parent = parent
		c.flags = fs
	}
	return nil
}

// AddSubCommand will add the given command as a sub-command of this command. An error is returned if the given command
// already has another parent.
func (c *Command) AddSubCommand(cmd *Command) error {
	if cmd.parent != nil {
		return fmt.Errorf("%w: %s", ErrCommandAlreadyHasParent, cmd.parent.name)
	}
	c.subCommands = append(c.subCommands, cmd)
	if err := cmd.setParent(c); err != nil {
		return fmt.Errorf("failed setting parent for command '%s': %w", cmd.name, err)
	}
	return nil
}

// inferCommandAndArgs takes the given CLI arguments, and splits them into flags, positional arguments, but most
// importantly, understands which command the user is trying to invoke. This is done by comparing given positional
// arguments to the current command hierarchy, and removing positional arguments that denote sub-commands.
//
// For example, assuming the following command line is given:
//
//	cmd1 -flag1 sub1 something -flag2=1 sub2 -- sub3 -flag3 a b c
//
// And the command hierarchy is: cmd1 -> sub1 -> sub2 -> sub3
//
// The returned values would be:
//   - flags: [-flag1, -flag2=1]: no "-flag3" because it's after the "--" separator
//   - positionals: [something, sub3, a, b, c]: no "cmd1", "sub1" and "sub2" as they are commands in the hierarchy
//   - command: sub2 (since it's the last valid command before the "--" which signals positional args only)
func (c *Command) inferCommandAndArgs(args []string) (flags, positionals []string, current *Command) {
	current = c
	onlyPositionalArgs := false
	for _, arg := range args {
		if onlyPositionalArgs {
			positionals = append(positionals, arg)
		} else if arg == "--" {
			onlyPositionalArgs = true
		} else if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
		} else {
			found := false
			for _, subCmd := range current.subCommands {
				if subCmd.name == arg {
					current = subCmd
					found = true
					break
				}
			}
			if !found {
				positionals = append(positionals, arg)
			}
		}
	}
	return
}

// getFullName returns the names of all commands in this command's hierarchy, starting from the root, all the way to
// this command.
//
// For example, assuming the following command hierarchy:
//
//	cmd1 -> sub1 -> sub2 -> sub3
//
// This function would return "cmd1 sub1" for the "sub1" command.
func (c *Command) getFullName() string {
	var fullName string
	for cmd := c; cmd != nil; cmd = cmd.parent {
		if fullName != "" {
			fullName = " " + fullName
		}
		fullName = cmd.name + fullName
	}
	return fullName
}

// getChain returns the chain of commands for this command, starting from the root, all the way to this command.
func (c *Command) getChain() []*Command {
	var chain []*Command
	for cmd := c; cmd != nil; cmd = cmd.parent {
		chain = append([]*Command{cmd}, chain...)
	}
	return chain
}

func (c *Command) PrintHelp(w io.Writer, width int) error {
	ww, err := NewWrappingWriter(width)
	if err != nil {
		return err
	}

	prefix4 := strings.Repeat(" ", 4)
	prefix8 := strings.Repeat(" ", 8)
	fullName := c.getFullName()

	// Command name & short description
	if c.shortDescription != "" {
		_, _ = fmt.Fprint(ww, fullName)
		_, _ = fmt.Fprint(ww, ": ")
		_ = ww.SetLinePrefix(prefix4)
		_, _ = fmt.Fprintln(ww, c.shortDescription)
		_ = ww.SetLinePrefix("")
	} else {
		_, _ = fmt.Fprintln(ww, fullName)
	}
	_, _ = fmt.Fprintln(ww)

	// Long description if we have one
	if c.longDescription != "" {
		_, _ = fmt.Fprint(ww, "Description: ")
		_ = ww.SetLinePrefix(prefix4)
		_, _ = fmt.Fprintln(ww, c.longDescription)
		_ = ww.SetLinePrefix("")
		_, _ = fmt.Fprintln(ww)
	}

	// Usage line
	_, _ = fmt.Fprintln(ww, "Usage:")
	_ = ww.SetLinePrefix(prefix4)
	_, _ = fmt.Fprint(ww, fullName+" ")
	_ = ww.SetLinePrefix(prefix8)
	if err := c.flags.printFlagsSingleLine(ww); err != nil {
		return err
	}
	_ = ww.SetLinePrefix("")
	_, _ = fmt.Fprintln(ww)
	_, _ = fmt.Fprintln(ww)

	// Flags
	if c.flags.hasFlags() {
		_, _ = fmt.Fprintln(ww, "Flags:")
		_ = ww.SetLinePrefix(prefix4)
		if err := c.flags.printFlagsMultiLine(ww, prefix4); err != nil {
			return err
		}
		_ = ww.SetLinePrefix("")
		_, _ = fmt.Fprintln(ww)
	}

	// Sub-commands
	if len(c.subCommands) > 0 {
		_, _ = fmt.Fprintln(ww, "Available sub-commands:")

		lenOfLongestSubCommand := 0
		for _, subCmd := range c.subCommands {
			if len(subCmd.name) > lenOfLongestSubCommand {
				lenOfLongestSubCommand = len(subCmd.name)
			}
		}
		subCommandNameDescSpacing := 10 - lenOfLongestSubCommand%10
		subCommandDescriptionCol := lenOfLongestSubCommand + subCommandNameDescSpacing

		for _, subCmd := range c.subCommands {
			_ = ww.SetLinePrefix(prefix4)
			_, _ = fmt.Fprint(ww, subCmd.name)
			_, _ = fmt.Fprint(ww, strings.Repeat(" ", subCommandDescriptionCol-len(subCmd.name)))
			_ = ww.SetLinePrefix(strings.Repeat(" ", len(prefix4)+subCommandDescriptionCol))
			_, _ = fmt.Fprintln(ww, subCmd.shortDescription)
		}
		_, _ = fmt.Fprintln(ww)

	}

	if _, err = w.Write([]byte(ww.String())); err != nil {
		return err
	}
	return nil
}

func (c *Command) PrintUsageLine(w io.Writer, width int) error {
	ww, err := NewWrappingWriter(width)
	if err != nil {
		return err
	}

	prefix4 := strings.Repeat(" ", 4)
	fullName := c.getFullName()

	_, _ = fmt.Fprint(ww, "Usage: ")
	_ = ww.SetLinePrefix(prefix4)
	_, _ = fmt.Fprint(ww, fullName+" ")
	if err := c.flags.printFlagsSingleLine(ww); err != nil {
		return err
	}
	_ = ww.SetLinePrefix("")
	_, _ = fmt.Fprintln(ww)

	if _, err = w.Write([]byte(ww.String())); err != nil {
		return err
	}
	return nil
}

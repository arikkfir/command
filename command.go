package command

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

//goland:noinspection GoUnusedGlobalVariable
var Version = "0.0.0-unknown"

var (
	tokenRE = regexp.MustCompile(`^([^=]+)=(.*)$`)
)

func New(parent *Command, spec Spec) *Command {
	if parent != nil && !parent.createdByNewCommand {
		panic("illegal parent was specified - was the parent created by 'command.New(...)'?")
	}
	cmd := &Command{
		Name:                spec.Name,
		ShortDescription:    spec.ShortDescription,
		LongDescription:     spec.LongDescription,
		Config:              spec.Config,
		PreSubCommandRun:    spec.OnSubCommandRun,
		Run:                 spec.Run,
		builtinConfig:       &BuiltinConfig{Help: false},
		parent:              parent,
		createdByNewCommand: true,
	}
	if cmd.parent != nil {
		cmd.parent.subCommands = append(cmd.parent.subCommands, cmd)
	}
	return cmd
}

type Spec struct {
	Name             string
	ShortDescription string
	LongDescription  string
	Config           any
	OnSubCommandRun  func(ctx context.Context, config any, usagePrinter UsagePrinter) error
	Run              func(ctx context.Context, config any, usagePrinter UsagePrinter) error
}

type Command struct {
	Name                 string
	ShortDescription     string
	LongDescription      string
	Config               any
	PreSubCommandRun     func(ctx context.Context, config any, usagePrinter UsagePrinter) error
	Run                  func(ctx context.Context, config any, usagePrinter UsagePrinter) error
	builtinConfig        any
	parent               *Command
	subCommands          []*Command
	createdByNewCommand  bool
	envVarsMapping       map[string]reflect.Value
	flagSet              *flag.FlagSet
	flagArgNames         map[string]string
	requiredFlags        []string
	positionalArgsTarget *[]string
}

type BuiltinConfig struct {
	Help bool `desc:"Show help about how to use this command"`
}

type UsagePrinter interface {
	PrintShortUsage(w io.Writer)
	PrintFullUsage(w io.Writer)
}

func (c *Command) PrintShortUsage(w io.Writer) {
	c.printCommandUsage(w, true)
}

func (c *Command) PrintFullUsage(w io.Writer) {
	c.printCommandUsage(w, false)
}

func (c *Command) initializeFlagSet() error {
	if c.flagArgNames == nil {
		c.flagArgNames = make(map[string]string)
	}

	// Create a flag set
	name := c.Name
	for parent := c.parent; parent != nil; parent = parent.parent {
		name = parent.Name + " " + name
	}
	c.flagSet = flag.NewFlagSet(name, flag.ContinueOnError)
	c.flagSet.SetOutput(io.Discard)
	if err := c.initializeFlagSetFromStruct(reflect.ValueOf(c.builtinConfig).Elem()); err != nil {
		return fmt.Errorf("failed to process builtin configuration fields: %w", err)
	}

	// If this command has no configuration stop here
	if c.Config == nil {
		return nil
	}

	// Verify the configuration field's type: must be a pointer to a struct, nothing else is accepted
	valueOfCfg := reflect.ValueOf(c.Config)
	if valueOfCfg.Kind() == reflect.Struct {
		return fmt.Errorf("field 'Config' in command '%s' must not be a struct, but a pointer to a struct: %+v", c.Name, c.Config)
	} else if valueOfCfg.Kind() != reflect.Ptr {
		return fmt.Errorf("field 'Config' in command '%s' is not a pointer: %+v", c.Name, c.Config)
	} else if valueOfCfg.IsNil() {
		return nil
	}
	valueOfCfgStruct := valueOfCfg.Elem()
	if valueOfCfgStruct.Kind() != reflect.Struct {
		return fmt.Errorf("field 'Config' in command '%s' is not a pointer to struct: %+v", c.Name, c.Config)
	}

	// Process the struct fields as flags
	if err := c.initializeFlagSetFromStruct(valueOfCfgStruct); err != nil {
		return fmt.Errorf("failed to process configuration fields: %w", err)
	}

	return nil
}

func (c *Command) initializeFlagSetFromStruct(valueOfCfgStruct reflect.Value) error {
	for i := 0; i < valueOfCfgStruct.NumField(); i++ {
		structField := valueOfCfgStruct.Type().Field(i)
		fieldName := structField.Name
		fieldValue := valueOfCfgStruct.Field(i)
		if !fieldValue.CanAddr() {
			return fmt.Errorf("field '%s' is not addressable", fieldName)
		} else if !fieldValue.CanSet() {
			return fmt.Errorf("field '%s' is not settable", fieldName)
		}

		flagName := fieldNameToFlagName(fieldName)
		description := structField.Tag.Get("desc")

		envVarName := fieldNameToEnvVarName(fieldName)
		if c.envVarsMapping == nil {
			c.envVarsMapping = make(map[string]reflect.Value)
		}

		// TODO: support commas inside token values (currently split will incorrectly split them)
		targetPtr := fieldValue.Addr().Interface()
		positionalsField := false
		for _, token := range strings.Split(structField.Tag.Get("flag"), ",") {
			if token == "ignore" {
				if slices.Contains(c.requiredFlags, flagName) {
					return fmt.Errorf("field '%s' cannot be both required and ignored", fieldName)
				} else {
					continue
				}
			} else if token == "required" {
				c.requiredFlags = append(c.requiredFlags, flagName)
			} else if token == "args" {
				if structField.Type.ConvertibleTo(reflect.TypeOf([]string{})) == false {
					return fmt.Errorf("field '%s' has 'args' tag but is not of type '[]string'", fieldName)
				} else if c.positionalArgsTarget != nil {
					return fmt.Errorf("multiple fields with 'args' tag found in command '%s'", c.Name)
				} else {
					c.positionalArgsTarget = targetPtr.(*[]string)
					positionalsField = true
				}
				continue
			} else if keyValue := tokenRE.FindStringSubmatch(token); keyValue != nil {
				key := keyValue[1]
				value := keyValue[2]
				switch key {
				case "valueName":
					c.flagArgNames[flagName] = value
				default:
					return fmt.Errorf("unsupported config tag key: %s", key)
				}
			}
		}
		if positionalsField == true {
			continue
		}

		switch fieldValue.Kind() {
		case reflect.Bool:
			c.flagSet.BoolVar(targetPtr.(*bool), flagName, fieldValue.Bool(), description)
			c.flagArgNames[flagName] = "" // to disable value name in usage page
			c.envVarsMapping[envVarName] = fieldValue
		case reflect.Int:
			c.flagSet.IntVar(targetPtr.(*int), flagName, int(fieldValue.Int()), description)
			c.envVarsMapping[envVarName] = fieldValue
		case reflect.Uint:
			c.flagSet.UintVar(targetPtr.(*uint), flagName, uint(fieldValue.Uint()), description)
			c.envVarsMapping[envVarName] = fieldValue
		case reflect.Float64:
			c.flagSet.Float64Var(targetPtr.(*float64), flagName, fieldValue.Float(), description)
			c.envVarsMapping[envVarName] = fieldValue
		case reflect.String:
			c.flagSet.StringVar(targetPtr.(*string), flagName, fieldValue.String(), description)
			c.envVarsMapping[envVarName] = fieldValue
		case reflect.Struct:
			if err := c.initializeFlagSetFromStruct(fieldValue); err != nil {
				return fmt.Errorf("failed adding flags for field '%s': %w", fieldName, err)
			}
		default:
			panic(fmt.Sprintf("unsupported configuration field type: %s\n", fieldValue.Kind()))
		}
	}

	return nil
}

func (c *Command) applyEnvironmentVariables(envVars map[string]string) error {
	if c.envVarsMapping != nil {
		for envVarName, fieldValue := range c.envVarsMapping {
			targetPtr := fieldValue.Addr().Interface()
			switch fieldValue.Kind() {
			case reflect.Bool:
				if stringValue, found := envVars[envVarName]; found {
					if boolValue, err := strconv.ParseBool(stringValue); err != nil {
						return fmt.Errorf("failed to parse environment variable '%s': %w", envVarName, err)
					} else {
						*targetPtr.(*bool) = boolValue
					}
				}
			case reflect.Int:
				if stringValue, found := envVars[envVarName]; found {
					if intValue, err := strconv.ParseInt(stringValue, 10, 0); err != nil {
						return fmt.Errorf("failed to parse environment variable '%s': %w", envVarName, err)
					} else {
						*targetPtr.(*int) = int(intValue)
					}
				}
			case reflect.Uint:
				if stringValue, found := envVars[envVarName]; found {
					if uintValue, err := strconv.ParseUint(stringValue, 10, 0); err != nil {
						return fmt.Errorf("failed to parse environment variable '%s': %w", envVarName, err)
					} else {
						*targetPtr.(*uint) = uint(uintValue)
					}
				}
			case reflect.Float64:
				if stringValue, found := envVars[envVarName]; found {
					if float64Value, err := strconv.ParseFloat(stringValue, 0); err != nil {
						return fmt.Errorf("failed to parse environment variable '%s': %w", envVarName, err)
					} else {
						*targetPtr.(*float64) = float64Value
					}
				}
			case reflect.String:
				if value, found := envVars[envVarName]; found {
					*targetPtr.(*string) = value
				}
			default:
				panic(fmt.Sprintf("unsupported configuration field type: %s\n", fieldValue.Kind()))
			}
		}
	}
	return nil
}

func (c *Command) applyCLIArguments(args []string) error {

	// Update config with CLI arguments
	if err := c.flagSet.Parse(args); err != nil {
		return fmt.Errorf("failed to apply CLI arguments: %w", err)
	}

	return nil
}

func (c *Command) validateRequiredFlagsWereProvided(envVars map[string]string) error {
	var missingRequiredFlags []string
	copy(missingRequiredFlags, c.requiredFlags)
	c.flagSet.Visit(func(f *flag.Flag) {
		missingRequiredFlags = slices.DeleteFunc(missingRequiredFlags, func(requiredFlagName string) bool {
			return requiredFlagName == f.Name
		})
	})
	for envVarName := range c.envVarsMapping {
		if _, found := envVars[envVarName]; found {
			envVarFlagName := environmentVariableToFlagName(envVarName)
			missingRequiredFlags = slices.DeleteFunc(missingRequiredFlags, func(requiredFlagName string) bool {
				return requiredFlagName == envVarFlagName
			})
		}
	}
	if len(missingRequiredFlags) > 0 {
		return fmt.Errorf("these required flags have not set via either CLI nor environment variables: %v", missingRequiredFlags)
	}
	return nil
}

func (c *Command) configure(envVars map[string]string, args []string) error {

	// Initialize the flagSet for the chosen command
	if err := c.initializeFlagSet(); err != nil {
		panic(fmt.Sprintf("failed to initialize flag set for command '%s': %v", c.Name, err))
	}

	// Apply environment variables first
	if err := c.applyEnvironmentVariables(envVars); err != nil {
		return fmt.Errorf("failed to apply environment variables: %w", err)
	}

	// Override with CLI arguments
	if err := c.flagSet.Parse(args); err != nil {
		return fmt.Errorf("failed to apply CLI arguments: %w", err)
	}

	// Apply positional arguments
	if c.positionalArgsTarget != nil {
		*c.positionalArgsTarget = c.flagSet.Args()
	}

	return nil
}

func (c *Command) printCommandUsage(w io.Writer, short bool) {
	cmdChain := c.Name
	for cmd := c.parent; cmd != nil; cmd = cmd.parent {
		cmdChain = cmd.Name + " " + cmdChain
	}

	if !short {
		_, _ = fmt.Fprintf(w, "%s: %s\n", cmdChain, c.ShortDescription)
		_, _ = fmt.Fprintln(w)
		if c.LongDescription != "" {
			_, _ = fmt.Fprintf(w, "%s\n", c.LongDescription)
			_, _ = fmt.Fprintln(w)
		}
	}

	flags := &bytes.Buffer{}
	lenOfLongestFlagName := 0
	c.flagSet.VisitAll(func(f *flag.Flag) {
		if len(f.Name) > lenOfLongestFlagName {
			lenOfLongestFlagName = len(f.Name)
		}
		_, _ = fmt.Fprint(flags, " ")
		required := slices.Contains(c.requiredFlags, f.Name)
		if !required {
			_, _ = fmt.Fprint(flags, "[")
		}
		if c.flagArgNames != nil {
			if valueName, ok := c.flagArgNames[f.Name]; ok {
				if valueName != "" {
					_, _ = fmt.Fprintf(flags, "--%s=%s", f.Name, c.flagArgNames[f.Name])
				} else {
					_, _ = fmt.Fprintf(flags, "--%s", f.Name)
				}
			} else {
				_, _ = fmt.Fprintf(flags, "--%s=VALUE", f.Name)
			}
		} else {
			_, _ = fmt.Fprintf(flags, "--%s=VALUE", f.Name)
		}
		if !required {
			_, _ = fmt.Fprint(flags, "]")
		}
	})
	positionalArgs := ""
	if c.positionalArgsTarget != nil {
		positionalArgs = " [ARGS]"
	}

	_, _ = fmt.Fprintf(w, "Usage:\n\t%s%s%s\n", cmdChain, flags, positionalArgs)
	_, _ = fmt.Fprintln(w)

	if !short {
		if lenOfLongestFlagName > 0 {
			var usageStartColumn int
			for usageStartColumn = 0; ; usageStartColumn += 10 {
				if usageStartColumn > lenOfLongestFlagName {
					break
				}
			}
			_, _ = fmt.Fprintf(w, "Flags:\n")
			c.flagSet.VisitAll(func(f *flag.Flag) {
				flagDesc := f.Usage
				if f.DefValue != "" {
					flagDesc += fmt.Sprintf(" (default is %s)", f.DefValue)
				}
				_, _ = fmt.Fprintf(w, "\t--%s%s%s\n", f.Name, strings.Repeat(" ", usageStartColumn-len(f.Name)), flagDesc)
			})
			_, _ = fmt.Fprintln(w)
		}

		if len(c.subCommands) > 0 {
			lenOfLongestSubcommandName := 0
			for _, subCmd := range c.subCommands {
				if len(subCmd.Name) > lenOfLongestSubcommandName {
					lenOfLongestSubcommandName = len(subCmd.Name)
				}
			}
			var usageStartColumn int
			for usageStartColumn = 0; ; usageStartColumn += 10 {
				if usageStartColumn > lenOfLongestSubcommandName {
					break
				}
			}
			_, _ = fmt.Fprintf(w, "Available sub-commands:\n")
			for _, subCmd := range c.subCommands {
				_, _ = fmt.Fprintf(w, "\t%s%s%s\n", subCmd.Name, strings.Repeat(" ", usageStartColumn-len(subCmd.Name)), subCmd.ShortDescription)
			}
			_, _ = fmt.Fprintln(w)
		}
	}
}

func Execute(ctx context.Context, w io.Writer, root *Command, args []string, envVars map[string]string) (exitCode int) {
	if !root.createdByNewCommand {
		panic("invalid root command given, indicating it may not have been created by 'command.New(...)'")
	}

	// Iterate CLI args, separate them to flags & positional args, but also infer the command to execute from the given
	// non-flags arguments; for example, assuming the following command line is given:
	//
	//		cmd1 -flag1 sub1 something -flag2=1 sub2 -- sub3 -flag3 a b c
	//
	// And the command hierarchy is: cmd1 -> sub1 -> sub2 -> sub3
	//
	// The returned values would be:
	//	command: sub2 (since it's the last valid command before the "--" which signals positional args only)
	//	flags: [-flag1, -flag2=1]: no "-flag3" because it's after the "--" separator
	//	positional args: [something, sub3, a, b, c]: no "cmd1", "sub1" and "sub2" as they are commands in the hierarchy
	cmd, flagArgs, positionalArgs := inferCommandFlagsAndPositionals(root, args)

	// Build the command chain from top-to-bottom (so index 0 is the root)
	commandChain := []*Command{cmd}
	parent := cmd.parent
	for parent != nil {
		commandChain = append([]*Command{parent}, commandChain...)
		parent = parent.parent
	}

	// Configure commands up the chain, in order to invoke their "PreSubCommandRun" function
	for _, current := range commandChain {
		if err := current.configure(envVars, append(flagArgs, positionalArgs...)); err != nil {
			current.PrintShortUsage(w)
			return 1
		}

		if current.PreSubCommandRun != nil {
			if err := current.PreSubCommandRun(ctx, current.Config, current); err != nil {
				_, _ = fmt.Fprintln(w, err.Error())
				return 1
			}
		}
	}

	// If "--help" was provided, show usage and exit immediately
	if cmd.flagSet.Lookup("help").Value.String() == "true" {
		cmd.PrintFullUsage(w)
		return 0
	}

	// If command has no "Run" function, it's an intermediate probably - just print its usage and exit successfully
	if cmd.Run == nil {
		cmd.PrintFullUsage(w)
		return 0
	}

	// Run the command
	if err := cmd.Run(ctx, cmd.Config, cmd); err != nil {
		_, _ = fmt.Fprintln(w, err.Error())
		return 1
	}

	return 0
}

package command

import (
	"fmt"
	"strings"
	"unicode"
)

func fieldNameToFlagName(fieldName string) string {
	var result []rune
	for i, r := range fieldName {
		if i == 0 {
			result = append(result, unicode.ToLower(r))
		} else if unicode.IsLower(r) {
			if i >= 2 && unicode.IsUpper(rune(fieldName[i-1])) && unicode.IsUpper(rune(fieldName[i-2])) {
				last := result[len(result)-1]
				result = append(result[0:len(result)-1], '-', last)
			}
			result = append(result, r)
		} else if unicode.IsUpper(r) {
			if unicode.IsLower(rune(fieldName[i-1])) {
				result = append(result, '-')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			panic(fmt.Sprintf("rune '%v' is neither uppercase nor lowercase", r))
		}
	}
	return string(result)
}

func fieldNameToEnvVarName(fieldName string) string {
	var result []rune
	for i, r := range fieldName {
		if i == 0 {
			result = append(result, unicode.ToUpper(r))
		} else if unicode.IsLower(r) {
			if i >= 2 && unicode.IsUpper(rune(fieldName[i-1])) && unicode.IsUpper(rune(fieldName[i-2])) {
				last := result[len(result)-1]
				result = append(result[0:len(result)-1], '_', last)
			}
			result = append(result, unicode.ToUpper(r))
		} else if unicode.IsUpper(r) {
			if unicode.IsLower(rune(fieldName[i-1])) {
				result = append(result, '_')
			}
			result = append(result, unicode.ToUpper(r))
		} else {
			panic(fmt.Sprintf("rune '%v' is neither uppercase nor lowercase", r))
		}
	}
	return string(result)
}

func environmentVariableToFlagName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}

func inferCommandFlagsAndPositionals(root *Command, args []string) (*Command, []string, []string) {
	var flagArgs []string
	var positionalArgs []string

	cmd := root
	onlyPositionalArgs := false
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if onlyPositionalArgs {
			positionalArgs = append(positionalArgs, arg)
		} else if arg == "--" {
			onlyPositionalArgs = true
		} else if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
		} else {
			found := false
			for _, subCmd := range cmd.subCommands {
				if subCmd.Name == arg {
					cmd = subCmd
					found = true
					break
				}
			}
			if !found {
				positionalArgs = append(positionalArgs, arg)
			}
		}
	}

	return cmd, flagArgs, positionalArgs
}

func EnvVarsArrayToMap(envVars []string) map[string]string {
	envVarsMap := make(map[string]string)
	for _, nameValue := range envVars {
		parts := strings.SplitN(nameValue, "=", 2)
		if len(parts) != 2 {
			panic(fmt.Sprintf("illegal environment variable: %s", nameValue))
		}
		envVarsMap[parts[0]] = parts[1]
	}
	return envVarsMap
}

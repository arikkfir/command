package command

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	"golang.org/x/sys/unix"
)

func ptrOf[T any](v T) *T {
	return &v
}

func defaultIfNil[T any](v *T, defaultValue T) T {
	if v == nil {
		return defaultValue
	}
	return *v
}

func intForBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

func fieldNameToFlagName(fieldName string) string {
	var result []rune
	for i, r := range fieldName {
		if i == 0 {
			result = append(result, unicode.ToLower(r))
		} else if unicode.IsUpper(r) {
			if unicode.IsLower(rune(fieldName[i-1])) {
				result = append(result, '-')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			if i >= 2 && unicode.IsUpper(rune(fieldName[i-1])) && unicode.IsUpper(rune(fieldName[i-2])) {
				last := result[len(result)-1]
				result = append(result[0:len(result)-1], '-', last)
			}
			result = append(result, r)
		}
	}
	return string(result)
}

func flagNameToEnvVarName(flagName string) string {
	return strings.ReplaceAll(strings.ToUpper(flagName), "-", "_")
}

func fieldNameToEnvVarName(fieldName string) string {
	var result []rune
	for i, r := range fieldName {
		if i == 0 {
			result = append(result, unicode.ToUpper(r))
		} else if unicode.IsUpper(r) {
			if unicode.IsLower(rune(fieldName[i-1])) {
				result = append(result, '_')
			}
			result = append(result, unicode.ToUpper(r))
		} else {
			if i >= 2 && unicode.IsUpper(rune(fieldName[i-1])) && unicode.IsUpper(rune(fieldName[i-2])) {
				last := result[len(result)-1]
				result = append(result[0:len(result)-1], '_', last)
			}
			result = append(result, unicode.ToUpper(r))
		}
	}
	return string(result)
}

//goland:noinspection GoUnusedExportedFunction
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

func getTerminalWidth() int {
	fd := int(os.Stdout.Fd())
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		return 80
	}
	return int(ws.Col)
}

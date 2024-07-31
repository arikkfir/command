package command

import (
	"cmp"
	"errors"
	"flag"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Tag string

const (
	TagFlag        Tag = "flag"
	TagName        Tag = "name"
	TagEnv         Tag = "env"
	TagValueName   Tag = "value-name"
	TagDescription Tag = "desc"
	TagRequired    Tag = "required"
	TagInherited   Tag = "inherited"
	TagArgs        Tag = "args"
)

type ErrInvalidTag struct {
	Cause error
	Tag   Tag
	Value string
}

func (e *ErrInvalidTag) Error() string {
	return fmt.Sprintf("invalid tag '%s=%s': %s", e.Tag, e.Value, e.Cause)
}

func (e *ErrInvalidTag) Unwrap() error {
	return e.Cause
}

type ErrUnknownFlag struct {
	Cause error
	Flag  string
}

func (e *ErrUnknownFlag) Error() string {
	return fmt.Sprintf("unknown flag: --%s", e.Flag)
}

func (e *ErrUnknownFlag) Unwrap() error {
	return e.Cause
}

type ErrRequiredFlagMissing struct {
	Cause error
	Flag  string
}

func (e *ErrRequiredFlagMissing) Error() string {
	return fmt.Sprintf("required flag is missing: --%s", e.Flag)
}

func (e *ErrRequiredFlagMissing) Unwrap() error {
	return e.Cause
}

type flagSet struct {
	flags              []*flagDef
	parent             *flagSet
	positionalsTargets []*[]string
}

func newFlagSet(parent *flagSet, objects ...reflect.Value) (*flagSet, error) {
	fs := &flagSet{parent: parent}
	for _, c := range objects {
		if c.Kind() == reflect.Ptr && c.Type().Elem().Kind() == reflect.Struct {
			if c.IsNil() {
				c.Set(reflect.New(c.Type().Elem()))
			}
			if err := fs.readFlagsFromStruct(c.Elem(), false); err != nil {
				return nil, err
			}
		}
	}
	return fs, nil
}

func (fs *flagSet) hasFlags() bool {
	if len(fs.flags) > 0 {
		return true
	}
	for _fs := fs.parent; _fs != nil; _fs = _fs.parent {
		for _, fd := range _fs.flags {
			if fd.Inherited {
				return true
			}
		}
	}
	return false
}

func (fs *flagSet) readFlagsFromStruct(s reflect.Value, defaultInherited bool) error {
	for i := 0; i < s.NumField(); i++ {
		fieldValue := s.Field(i)
		structField := s.Type().Field(i)
		fieldName := structField.Name
		if err := fs.readFlagFromField(fieldValue, structField, defaultInherited); err != nil {
			return fmt.Errorf("invalid field '%s.%s': %w", s.Type(), fieldName, err)
		}
	}
	return nil
}

func (fs *flagSet) readFlagFromField(fieldValue reflect.Value, structField reflect.StructField, defaultInherited bool) error {
	fieldName := structField.Name

	// Initial configuration of this field
	var args bool
	var flagTag Tag
	fd := &flagDef{
		flagInfo:  flagInfo{Name: fieldNameToFlagName(fieldName)},
		Inherited: defaultInherited,
		Targets:   []reflect.Value{fieldValue},
	}

	// Read field tags
	if tag, ok := structField.Tag.Lookup(string(TagFlag)); ok {
		if v, err := strconv.ParseBool(tag); err != nil {
			var ne *strconv.NumError
			if errors.As(err, &ne) {
				err = ne.Err
			}
			return &ErrInvalidTag{Cause: err, Tag: TagFlag, Value: tag}
		} else if !v {
			return nil
		} else {
			flagTag = TagFlag
		}
	}
	if tag, ok := structField.Tag.Lookup(string(TagName)); ok {
		if tag == "" {
			return &ErrInvalidTag{Cause: fmt.Errorf("must not be empty"), Tag: TagName, Value: tag}
		}
		flagTag = TagName
		fd.flagInfo.Name = tag
	}
	if tag, ok := structField.Tag.Lookup(string(TagEnv)); ok {
		if tag == "" {
			return &ErrInvalidTag{Cause: fmt.Errorf("must not be empty"), Tag: TagEnv, Value: tag}
		} else {
			tag = strings.ToUpper(tag)
		}
		flagTag = TagEnv
		fd.flagInfo.EnvVarName = &tag
	}
	if tag, ok := structField.Tag.Lookup(string(TagValueName)); ok {
		if tag == "" {
			return &ErrInvalidTag{Cause: fmt.Errorf("must not be empty"), Tag: TagValueName, Value: tag}
		} else if fieldValue.Kind() == reflect.Bool {
			return &ErrInvalidTag{Cause: fmt.Errorf("not supported for bool fields"), Tag: TagValueName, Value: tag}
		}
		flagTag = TagValueName
		fd.flagInfo.ValueName = &tag
	}
	if tag, ok := structField.Tag.Lookup(string(TagDescription)); ok {
		flagTag = TagDescription
		fd.flagInfo.Description = &tag
	}
	if tag, ok := structField.Tag.Lookup(string(TagRequired)); ok {
		if v, err := strconv.ParseBool(tag); err != nil {
			var ne *strconv.NumError
			if errors.As(err, &ne) {
				err = ne.Err
			}
			return &ErrInvalidTag{Cause: err, Tag: TagRequired, Value: tag}
		} else {
			flagTag = TagRequired
			fd.flagInfo.Required = ptrOf(v)
		}
	}
	if tag, ok := structField.Tag.Lookup(string(TagInherited)); ok {
		if v, err := strconv.ParseBool(tag); err != nil {
			var ne *strconv.NumError
			if errors.As(err, &ne) {
				err = ne.Err
			}
			return &ErrInvalidTag{Cause: err, Tag: TagInherited, Value: tag}
		} else {
			flagTag = TagInherited
			fd.Inherited = v
		}
	}
	if tag, ok := structField.Tag.Lookup(string(TagArgs)); ok {
		if v, err := strconv.ParseBool(tag); err != nil {
			var ne *strconv.NumError
			if errors.As(err, &ne) {
				err = ne.Err
			}
			return &ErrInvalidTag{Cause: err, Tag: TagArgs, Value: tag}
		} else {
			args = v
		}
	}

	if fieldValue.Kind() == reflect.Struct {
		// Struct fields are only containers for other fields; if the struct is tagged with "args" or any flag tag, fail
		if args {
			return &ErrInvalidTag{Cause: fmt.Errorf("cannot be used on struct fields"), Tag: TagArgs, Value: strconv.FormatBool(args)}
		} else if flagTag != "" {
			return &ErrInvalidTag{Cause: fmt.Errorf("cannot be used on struct fields"), Tag: flagTag, Value: structField.Tag.Get(string(flagTag))}
		} else if err := fs.readFlagsFromStruct(fieldValue, fd.Inherited); err != nil {
			return err
		} else {
			return nil
		}
	} else if !args && flagTag == "" {
		// Neither a positional args target nor a flag - do nothing and exit
		return nil
	} else if !fieldValue.CanAddr() {
		// Field must be addressable or we will not be able to update it with CLI arguments
		return fmt.Errorf("not addressable")
	} else if !fieldValue.CanSet() {
		// Field must be settable or we will not be able to update it with CLI arguments
		return fmt.Errorf("not settable")
	} else if args {
		// If field is tagged with "args", it cannot also serve as a flag; it also must be of type "[]string"
		if flagTag != "" {
			return &ErrInvalidTag{Cause: fmt.Errorf("cannot be a flag as well"), Tag: TagArgs, Value: strconv.FormatBool(args)}
		} else if structField.Type.ConvertibleTo(reflect.TypeOf([]string{})) {
			fs.positionalsTargets = append(fs.positionalsTargets, fieldValue.Addr().Interface().(*[]string))
			return nil
		} else {
			return &ErrInvalidTag{Cause: fmt.Errorf("must be typed as []string"), Tag: TagArgs, Value: strconv.FormatBool(args)}
		}
	}

	// Configure whether flag should be given a value in the CLI, and the default value if one is not provided
	switch fieldValue.Kind() {
	case reflect.Bool:
		fd.HasValue = false
		fd.DefaultValue = "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fd.HasValue = true
		fd.DefaultValue = strconv.FormatInt(fieldValue.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		fd.HasValue = true
		fd.DefaultValue = strconv.FormatUint(fieldValue.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		fd.HasValue = true
		fd.DefaultValue = strconv.FormatFloat(fieldValue.Float(), 'g', -1, 64)
	case reflect.String:
		fd.HasValue = true
		fd.DefaultValue = fieldValue.String()
	case reflect.Slice:
		fd.HasValue = true
		var defaultValues []string
		for i := 0; i < fieldValue.Len(); i++ {
			defaultValues = append(defaultValues, fieldValue.Index(i).String())
		}
		if defaultValues != nil {
			fd.DefaultValue = strings.Join(defaultValues, ",")
		} else {
			fd.DefaultValue = ""
		}
	default:
		// Unsupported flag field type
		return fmt.Errorf("unsupported field type: %s", fieldValue.Kind())
	}

	// Otherwise, this is a flag - check if it has already been registered?
	for _, fdi := range fs.flags {
		if fdi.Name == fd.Name {
			if fdi.EnvVarName == nil {
				fdi.EnvVarName = fd.EnvVarName
			} else if fd.EnvVarName != nil && *fdi.EnvVarName != *fd.EnvVarName {
				return &ErrInvalidTag{Cause: fmt.Errorf("cannot redefine environment variable name"), Tag: TagEnv, Value: *fd.EnvVarName}
			}
			if fdi.HasValue != fd.HasValue {
				return fmt.Errorf("incompatible field types detected (is one a bool and another isn't?)")
			}
			if fdi.ValueName == nil {
				fdi.ValueName = fd.ValueName
			} else if fd.ValueName != nil && *fdi.ValueName != *fd.ValueName {
				return &ErrInvalidTag{Cause: fmt.Errorf("cannot redefine value name"), Tag: TagValueName, Value: *fd.ValueName}
			}
			if fdi.Description == nil {
				fdi.Description = fd.Description
			} else if fd.Description != nil && *fdi.Description != *fd.Description {
				return &ErrInvalidTag{Cause: fmt.Errorf("cannot redefine description"), Tag: TagDescription, Value: *fd.Description}
			}
			if fdi.Required == nil {
				fdi.Required = fd.Required
			} else if fd.Required != nil && *fdi.Required != *fd.Required {
				return &ErrInvalidTag{Cause: fmt.Errorf("cannot redefine required status"), Tag: TagRequired, Value: strconv.FormatBool(*fd.Required)}
			}
			if fdi.DefaultValue != fd.DefaultValue {
				return fmt.Errorf("incompatible default values detected: '%s' vs '%s'", fdi.DefaultValue, fd.DefaultValue)
			}
			if fdi.Inherited != fd.Inherited {
				return fmt.Errorf("incompatible inherited status detected: '%v' vs '%v'", fdi.Inherited, fd.Inherited)
			}
			fdi.Targets = append(fdi.Targets, fd.Targets...)
			return nil
		}
	}

	// New flag, add it as is
	fs.flags = append(fs.flags, fd)
	return nil
}

func (fs *flagSet) getMergedFlagDefs() ([]*mergedFlagDef, error) {
	flags := make(map[string]*mergedFlagDef)
	for cfs := fs; cfs != nil; cfs = cfs.parent {
		for _, fd := range cfs.flags {
			if cfs == fs || fd.Inherited {
				if mfd, ok := flags[fd.Name]; !ok {
					flags[fd.Name] = &mergedFlagDef{
						flagInfo: flagInfo{
							Name:         fd.Name,
							EnvVarName:   fd.EnvVarName,
							HasValue:     fd.HasValue,
							ValueName:    fd.ValueName,
							Description:  fd.Description,
							Required:     fd.Required,
							DefaultValue: fd.DefaultValue,
						},
						applied:  false,
						flagDefs: []*flagDef{fd},
					}
				} else if err := mfd.addFlagDef(fd); err != nil {
					return nil, err
				}
			}
		}
	}
	var mergedFlagDefs []*mergedFlagDef
	for _, mfd := range flags {
		if mfd.EnvVarName == nil {
			mfd.EnvVarName = ptrOf(flagNameToEnvVarName(mfd.Name))
		}
		if mfd.ValueName == nil {
			mfd.ValueName = ptrOf("VALUE")
		}
		if mfd.Required == nil {
			mfd.Required = ptrOf(false)
		}
		sort.Slice(mfd.flagDefs, func(ai, bi int) bool { return mfd.flagDefs[ai].isLessThan(mfd.flagDefs[bi]) })
		mergedFlagDefs = append(mergedFlagDefs, mfd)
	}
	sort.Slice(mergedFlagDefs, func(ai, bi int) bool { return cmp.Less(mergedFlagDefs[ai].Name, mergedFlagDefs[bi].Name) })
	return mergedFlagDefs, nil
}

func (fs *flagSet) apply(envVars map[string]string, args []string) error {
	if args == nil {
		args = []string{}
	}
	if envVars == nil {
		envVars = make(map[string]string)
	}

	stdFs := flag.NewFlagSet("", flag.ContinueOnError)
	stdFs.SetOutput(io.Discard)

	// Merge flags from this flag set and its parents
	mergedFlagDefs, err := fs.getMergedFlagDefs()
	if err != nil {
		return err
	}

	// Iterate flags and define them in the stdlib FlagSet
	for _, mfd := range mergedFlagDefs {

		// By definition, for the same name - all flags have the same "HasValue" value, so it should be safe to just
		// take it from the first one
		if mfd.HasValue {
			stdFs.Func(mfd.Name, "", func(v string) error { return mfd.setValue(v) })
		} else {
			stdFs.BoolFunc(mfd.Name, "", func(string) error { return mfd.setValue("true") })
		}

		// Set the field's default value so it's marked as "applied" (and thus the "required" validation will ignore it)
		if mfd.DefaultValue != "" {
			if err := mfd.setValue(mfd.DefaultValue); err != nil {
				return fmt.Errorf("failed applying default value for flag '%s': %w", mfd.Name, err)
			}
		}

		// Set the value to the flag's corresponding environment variable, if one was given
		// Important this is done here, so it overrides the default value set earlier
		if v, found := envVars[*mfd.EnvVarName]; found {
			if err := mfd.setValue(v); err != nil {
				return err
			}
		}
	}

	// Parse the given arguments, which will result in all CLI flags being set
	if err := stdFs.Parse(args); err != nil {
		re := regexp.MustCompile(`^flag provided but not defined: -(.+)$`)
		if matches := re.FindStringSubmatch(err.Error()); matches != nil {
			return &ErrUnknownFlag{Cause: err, Flag: matches[1]}
		}
		return err
	}

	// Verify all required flags have been set
	for _, mfd := range mergedFlagDefs {
		if mfd.isMissing() {
			return &ErrRequiredFlagMissing{Cause: err, Flag: mfd.Name}
		}
	}

	// Apply positionals
	positionals := stdFs.Args()
	for cfs := fs; cfs != nil; cfs = cfs.parent {
		for _, target := range cfs.positionalsTargets {
			*target = positionals
		}
	}
	return nil
}

func (fs *flagSet) printFlagsSingleLine(b io.Writer) error {

	// Merge flags from this flag set and its parents
	mergedFlagDefs, err := fs.getMergedFlagDefs()
	if err != nil {
		return err
	}

	space := false
	for _, fd := range mergedFlagDefs {
		if space {
			_, _ = fmt.Fprint(b, " ")
		} else {
			space = true
		}
		if !fd.isRequired() {
			_, _ = fmt.Fprint(b, "[")
		}

		valueName := fd.getValueName()
		if valueName != "" {
			_, _ = fmt.Fprintf(b, "--%s=%s", fd.Name, valueName)
		} else {
			_, _ = fmt.Fprintf(b, "--%s", fd.Name)
		}
		if !fd.isRequired() {
			_, _ = fmt.Fprint(b, "]")
		}
	}
	if len(fs.positionalsTargets) > 0 {
		if space {
			_, _ = fmt.Fprint(b, " ")
		}
		_, _ = fmt.Fprint(b, "[ARGS...]")
	}

	return nil
}

func (fs *flagSet) printFlagsMultiLine(ww *WrappingWriter, basePrefix string) error {

	// Merge flags from this flag set and its parents
	mergedFlagDefs, err := fs.getMergedFlagDefs()
	if err != nil {
		return err
	}

	flagsColWidth := 0
	fullFlagNames := make(map[string]string)
	for _, fd := range mergedFlagDefs {
		var fullFlagName string
		valueName := fd.getValueName()
		if valueName != "" {
			fullFlagName = fmt.Sprintf("--%s=%s", fd.Name, valueName)
		} else {
			fullFlagName = fmt.Sprintf("--%s", fd.Name)
		}
		if fd.Required == nil || !*fd.Required {
			fullFlagName = "[" + fullFlagName + "]"
		}
		fullFlagNames[fd.Name] = fullFlagName
		if len(fullFlagName) > flagsColWidth {
			flagsColWidth = len(fullFlagName)
		}
	}

	descriptionStartColumn := flagsColWidth + (10 - flagsColWidth%10)
	for _, fd := range mergedFlagDefs {
		flagName := fullFlagNames[fd.Name]
		_, _ = fmt.Fprint(ww, flagName)
		_, _ = fmt.Fprint(ww, strings.Repeat(" ", descriptionStartColumn-len(flagName)))
		_ = ww.SetLinePrefix(basePrefix + strings.Repeat(" ", descriptionStartColumn))

		// Build flag description
		hasDescription := fd.Description != nil && *fd.Description != ""
		var sep string
		if hasDescription {
			_, _ = fmt.Fprint(ww, *fd.Description)
			sep = " ("
		}

		if fd.DefaultValue != "" {
			if sep != "" {
				_, _ = fmt.Fprint(ww, sep)
			}
			_, _ = fmt.Fprintf(ww, "default value: %s", fd.DefaultValue)
			sep = ", "
		}
		if fd.EnvVarName != nil {
			if sep != "" {
				_, _ = fmt.Fprint(ww, sep)
			}
			_, _ = fmt.Fprintf(ww, "environment variable: %s", *fd.EnvVarName)
		}
		if hasDescription {
			_, _ = fmt.Fprint(ww, ")")
		}

		_ = ww.SetLinePrefix(basePrefix)
		_, _ = fmt.Fprintln(ww)
	}

	return nil
}

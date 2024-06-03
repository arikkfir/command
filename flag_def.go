package command

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

type ErrInvalidValue struct {
	Cause error
	Value string
	Flag  string
}

func (e *ErrInvalidValue) Error() string {
	return fmt.Sprintf("invalid value '%s' for flag '%s': %s", e.Value, e.Flag, e.Cause)
}

func (e *ErrInvalidValue) Unwrap() error {
	return e.Cause
}

type flagInfo struct {
	Name         string
	EnvVarName   *string
	HasValue     bool
	ValueName    *string
	Description  *string
	Required     *bool
	DefaultValue string
}

type flagDef struct {
	flagInfo
	Inherited bool
	Targets   []reflect.Value
	applied   bool
}

func (fd *flagDef) isRequired() bool {
	return fd.Required != nil && *fd.Required
}

func (fd *flagDef) getValueName() string {
	if fd.HasValue {
		if fd.ValueName != nil {
			return *fd.ValueName
		} else {
			return "VALUE"
		}
	} else {
		return ""
	}
}

func (fd *flagDef) setValue(sv string) error {
	for _, fv := range fd.Targets {
		switch fv.Kind() {
		case reflect.Bool:
			if b, err := strconv.ParseBool(sv); err != nil {
				var ne *strconv.NumError
				if errors.As(err, &ne) {
					return &ErrInvalidValue{Cause: ne.Err, Value: ne.Num, Flag: fd.Name}
				} else {
					return &ErrInvalidValue{Cause: err, Value: sv, Flag: fd.Name}
				}
			} else {
				fv.SetBool(b)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if i, err := strconv.ParseInt(sv, 10, 64); err != nil {
				var ne *strconv.NumError
				if errors.As(err, &ne) {
					return &ErrInvalidValue{Cause: ne.Err, Value: ne.Num, Flag: fd.Name}
				} else {
					return &ErrInvalidValue{Cause: err, Value: sv, Flag: fd.Name}
				}
			} else {
				fv.SetInt(i)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if ui, err := strconv.ParseUint(sv, 10, 64); err != nil {
				var ne *strconv.NumError
				if errors.As(err, &ne) {
					return &ErrInvalidValue{Cause: ne.Err, Value: ne.Num, Flag: fd.Name}
				} else {
					return &ErrInvalidValue{Cause: err, Value: sv, Flag: fd.Name}
				}
			} else {
				fv.SetUint(ui)
			}
		case reflect.Float32, reflect.Float64:
			if f, err := strconv.ParseFloat(sv, 64); err != nil {
				var ne *strconv.NumError
				if errors.As(err, &ne) {
					return &ErrInvalidValue{Cause: ne.Err, Value: ne.Num, Flag: fd.Name}
				} else {
					return &ErrInvalidValue{Cause: err, Value: sv, Flag: fd.Name}
				}
			} else {
				fv.SetFloat(f)
			}
		case reflect.String:
			fv.SetString(sv)
		default:
			return fmt.Errorf("%w: field kind is '%s'", errors.ErrUnsupported, fv.Kind())
		}
	}
	fd.applied = true
	return nil
}

func (fd *flagDef) isLessThan(b *flagDef) bool {
	a := fd
	name := cmp.Compare(a.Name, b.Name)
	if name < 0 {
		return true
	} else if name > 0 {
		return false
	}
	envVarName := cmp.Compare(defaultIfNil(a.EnvVarName, ""), defaultIfNil(b.EnvVarName, ""))
	if envVarName < 0 {
		return true
	} else if envVarName > 0 {
		return false
	}
	hasValue := cmp.Compare(intForBool(a.HasValue), intForBool(b.HasValue))
	if hasValue < 0 {
		return true
	} else if hasValue > 0 {
		return false
	}
	valueName := cmp.Compare(defaultIfNil(a.ValueName, ""), defaultIfNil(b.ValueName, ""))
	if valueName < 0 {
		return true
	} else if valueName > 0 {
		return false
	}
	description := cmp.Compare(defaultIfNil(a.Description, ""), defaultIfNil(b.Description, ""))
	if description < 0 {
		return true
	} else if description > 0 {
		return false
	}
	required := cmp.Compare(intForBool(defaultIfNil(a.Required, false)), intForBool(defaultIfNil(b.Required, false)))
	if required < 0 {
		return true
	} else if required > 0 {
		return false
	}
	defaultValue := cmp.Compare(a.DefaultValue, b.DefaultValue)
	if defaultValue < 0 {
		return true
	} else if defaultValue > 0 {
		return false
	}
	inherited := cmp.Compare(intForBool(a.Inherited), intForBool(b.Inherited))
	if inherited < 0 {
		return true
	} else if inherited > 0 {
		return false
	}
	return false
}

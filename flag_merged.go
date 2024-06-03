package command

import (
	"fmt"
)

type mergedFlagDef struct {
	flagInfo
	applied  bool
	flagDefs []*flagDef
}

func (mfd *mergedFlagDef) addFlagDef(fd *flagDef) error {
	if fd.Name != mfd.Name {
		return fmt.Errorf("given flag '%s' has incompatible name - must be '%s'", fd.Name, mfd.Name)
	}

	if mfd.EnvVarName == nil {
		if fd.EnvVarName != nil {
			mfd.EnvVarName = fd.EnvVarName
		}
	} else if fd.EnvVarName != nil {
		if *mfd.EnvVarName != *fd.EnvVarName {
			return fmt.Errorf("flag '%s' has incompatible environment variable name '%v' - must be '%v'", fd.Name, *fd.EnvVarName, *mfd.EnvVarName)
		}
	}

	if fd.HasValue != mfd.HasValue {
		if mfd.HasValue {
			return fmt.Errorf("given flag '%s' must have a value, but it does not", fd.Name)
		} else {
			return fmt.Errorf("given flag '%s' must not have a value, but it does", fd.Name)
		}
	}

	if mfd.ValueName == nil {
		if fd.ValueName != nil {
			mfd.ValueName = fd.ValueName
		}
	} else if fd.ValueName != nil {
		if *mfd.ValueName != *fd.ValueName {
			return fmt.Errorf("flag '%s' has incompatible value-name '%v' - must be '%v'", fd.Name, *fd.ValueName, *mfd.ValueName)
		}
	}

	if mfd.Description == nil {
		if fd.Description != nil {
			mfd.Description = fd.Description
		}
	} else if fd.Description != nil {
		if *mfd.Description != *fd.Description {
			return fmt.Errorf("flag '%s' has incompatible description", fd.Name)
		}
	}

	if mfd.Required == nil {
		if fd.Required != nil {
			mfd.Required = fd.Required
		}
	} else if *mfd.Required {
		if fd.Required != nil && !*fd.Required {
			return fmt.Errorf("flag '%s' is incompatibly optional - must be required", fd.Name)
		}
	}

	if fd.DefaultValue != mfd.DefaultValue {
		return fmt.Errorf("flag '%s' has incompatible default value '%s' - must be '%s'", fd.Name, fd.DefaultValue, mfd.DefaultValue)
	}

	mfd.flagDefs = append(mfd.flagDefs, fd)
	return nil
}

func (mfd *mergedFlagDef) setValue(v string) error {
	mfd.applied = true
	for _, fd := range mfd.flagDefs {
		if err := fd.setValue(v); err != nil {
			return err
		}
	}
	return nil
}

func (mfd *mergedFlagDef) isRequired() bool {
	return mfd.Required != nil && *mfd.Required
}

func (mfd *mergedFlagDef) isMissing() bool {
	return mfd.isRequired() && !mfd.applied
}

func (mfd *mergedFlagDef) getValueName() string {
	if mfd.HasValue {
		if mfd.ValueName != nil {
			return *mfd.ValueName
		} else {
			return "VALUE"
		}
	} else {
		return ""
	}
}

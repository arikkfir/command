# command

![Maintainer](https://img.shields.io/badge/maintainer-arikkfir-blue)
![GoVersion](https://img.shields.io/github/go-mod/go-version/arikkfir/command.svg)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/arikkfir/command)
[![GoReportCard](https://goreportcard.com/badge/github.com/arikkfir/command)](https://goreportcard.com/report/github.com/arikkfir/command)

> CLI Command-based framework with extra sugar!

This small framework is intended to help in creating simple CLI applications with a hierarchical command-based,
with built-in configuration conforming to the 12-factor app manifest, yet without requiring the developer to write a lot
of boilerplate code.

It has the following goals:

* Utilize & play nice with the builtin Go `flag` package
* Builtin usage & help screens
* Hierarchical command structure
* Easy & simple (though opinionated) configuration facility

## Attribution

This library is **heavily** inspired by [Cobra](https://github.com/spf13/cobra), an **excellent** command framework.
I do feel, however, that it is a bit too heavy weight for just allowing users to write `main` CLI programs yet with the
familiar (`kubectl`-like) command hierarchy and easy configuration. Cobra, unfortunately, carries much more water -
for good and worse. It is packed with features and facilities which I believe most programs do not really need.

That said, big respect goes to `spf13` for creating that library - it is really exceptional, and has garnered a huge
community behind it. If you're looking for an all-in-one solution for extensive configuration, hooks, code generators, 
etc - Cobra is my recommendation.

## Usage

Create the following file structure:

```
demo
 |
 +-- main.go
 |
 +-- root.go
 |
 +-- command1.go
 |
 +-- command2.go
 |
 ...
```

For `root.go`, use this:

```go
package main

import (
	"os"
	"path/filepath"

	"github.com/arikkfir/command"
)

type RootConfig struct{
	SomeFlag string `desc:"This flag is a demo flag of type string."` // We can provide flag descriptions
}

var rootCommand = command.New(nil, command.Spec{
	Name:             filepath.Base(os.Args[0]),
	ShortDescription: "This is the root command.",
	LongDescription: `This is the command executed when no sub-commands are specified in the command line, e.g. like
running "kubectl" and pressing ENTER.`,
	Config: &RootConfig{
		SomeFlag1: "default value for this flag, unless given in CLI via --some-flag or via environment variables as SOME_FLAG",
	},
})
```

For `command1.go` use this:

```go
package main

import (
	"context"
	"fmt"

	"github.com/arikkfir/command"
)

type Command1Config struct {
	RootConfig // Root command's configuration can also be provided to this command (e.g. "--some-flag")
	AnotherFlag int `config:"required" desc:"This is another flag, of type int."` // Notice how we made this flag required
}

var cmd1Command = command.New(rootCommand, command.Spec{
	Name:             "command1",
	ShortDescription: "This is command1, a magnificent command that does something.",
	LongDescription:  `Longer description...`,
	Config:           &Command1Config{
		RootConfig: RootConfig{
			SomeFlag: "override the default value",
		},
		AnotherFlag: "default for AnotherFlag", // set in CLI as "--another-flag" or environment variable ANOTHER_FLAG
	},
	Run: func(ctx context.Context, anyConfig any, utils command.UsagePrinter) error {
		config := anyConfig.(*Command1Config)
		fmt.Printf("Running command1! Configuration is: %+v\n", config)
		return nil
	},
})
```

For `command2.go` use this:

```go
package main

import (
	"context"
	"fmt"

	"github.com/arikkfir/command"
)

type Command2Config struct {
	Command1Config
	YetAnotherFlag string `desc:"And another one..."`
	Positionals []string  `config:"args"` // This field will get all the non-flag positional arguments for the command
}

var cmd2Command = command.New(cmd1Command, command.Spec{
	Name:             "command2",
	ShortDescription: "This is command2, another magnificent command that does something.",
	Config:           &Command1Config{
		Command1Config: Command1Config{
			RootConfig: RootConfig{
				SomeFlag: "override the default value",
			},
			AnotherFlag: "default value", 
		},
		YetAnotherFlag: "default for YetAnotherFlag",
	},
	Run: func(ctx context.Context, anyConfig any, utils command.UsagePrinter) error {
		config := anyConfig.(*Command2Config)
		fmt.Printf("Running command2! Configuration is: %+v\n", config)
		return nil
	},
})
```

And finally create `main.go` like so:

```go
package main

import (
	"os"

	"github.com/arikkfir/command"
)

func main() {
	command.Execute(rootCommand, os.Args, command.EnvVarsArrayToMap(os.Environ()))
}
```


## Running

Once your program is compiled to a binary, you can run it like so:

```shell
$ myprogram --some-flag=someValue # this will run the root command; since no "Run" function was provided, it will print the usage screen
$ myprogram --some-flag=someValue command1 # runs "command1" with the flag from root, and the default value for "AnotherFlag"
$ myprogram --some-flag=someValue command1 --another-flag=anotherValue # runs "command1" with the flag from root, and the value for "AnotherFlag"
$ myprogram command1 --another-flag=anotherValue # runs "command1" with the default value for the root flag, and the value for "AnotherFlag"
$ myprogram command1 command2 # runs the "command2" command
```

## Usage & Help screens

For the root command (just running `myprogram`), this would be the usage page:

```go
$ myprogram --help
myprogram: This is the root command.

This is the command executed when no sub-commands are specified in the command line, e.g. like
running "kubectl" and pressing ENTER.

Usage:
	myprogram [--some-flag] [--help]

Flags:
	--some-flag    This flag is a demo flag of type string.
	--help         Print usage information (default is false)

Sub-commands:
	command1       This is command1, a magnificent command that does something.
```

For `command1`, this would be the usage page:

```go
$ myprogram command1 --help
myprogram command1: This is command1, a magnificent command that does something.

Longer description...

Usage:
	myprogram command1 [--some-flag] --another-flag [--help]

Flags:
	--some-flag    This flag is a demo flag of type string.
	--another-flag This is another flag, of type int.
	--help         Print usage information (default is false)

Sub-commands:
    command2       This is command2, another magnificent command that does something.
```

For `command2`, notice that since it doesn't have a long description set, nor any sub-commands, none are printed: 

```go
$ myprogram command1 command2 --help
myprogram command1 command2: This is command2, another magnificent command that does something.

Usage:
	myprogram command1 [--some-flag] --another-flag [--yet-another-flag] [--help] [ARGS]

Flags:
	--some-flag         This flag is a demo flag of type string.
	--another-flag      This is another flag, of type int.
	--yet-another-flag  And another one...
	--help              Print usage information (default is false)
```

## Naming of flags & environment variables

Fields in command configuration structs should be named in standard Go pascal-case (`MyField`). 

Flags for fields will be generated, with flag names as kebab-case (`--my-field`). 

Environment variables will be generated as an upper-case snake-case (`MY_FIELD`).

## Field tags

You can use Go tags for the configuration fields:

```go
package main

type MyConfig struct {
	RequiredField string `config:"required" desc:"This is a required field."`
	IgnoredField string  `config:"ignore"` // No flags will be generated for this field and it is not configurable via environment variables
	Args []string        `config:"args"` // This field will get all the non-flag positional arguments for the command
}
```

## Field types

Configuration fields cab be of type `string`, `int`, `uint`, `float64`, `bool`, or a `struct` containing additional
flags. When providing configuration in the command spec, **you must provide a pointer to the configuration struct**.

New types will be added soon (e.g. `time.Time`, `time.Duration`, `net.IP`, and more).

## Contributing

Please do :ok_hand: :muscle: !

See [CONTRIBUTING.md](CONTRIBUTING.md) for more information :pray:

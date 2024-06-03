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

For `command2.go` use this:

```go
package main

import (
	"context"
	"fmt"

	"github.com/arikkfir/command"
)

type Command2 struct {
	MyFlag1 string `flag:"true"`
	MyFlag2 int    `desc:"mf2" required:"true"`
}

func (c *Command2) PreRun(ctx context.Context) error {
	// Invoked every time this command or any of its sub-commands are run
	fmt.Println(c.MyFlag1)
	return nil
}

func (c *Command2) Run(ctx context.Context) error {
	// Invoked when this command is run
	fmt.Println(c.MyFlag2)
	return nil
}

var cmd2 = command.MustNew(
	"command2",
	"This is command2, a magnificent command that does something.",
	`Longer description...`,
	&Command2{
		MyFlag1: "default value for --my-flag1",
	},
)
```

For `command1.go` use this:

```go
package main

import (
	"context"
	"fmt"

	"github.com/arikkfir/command"
)

type Command1 struct {
	AnotherFlag bool   `flag:"true"`
	MyURL       string `value-name:"URL" env:"HTTP_URL"`
}

func (c *Command1) PreRun(ctx context.Context) error {
	// Invoked every time this command or any of its sub-commands are run
	fmt.Println(c.AnotherFlag)
	return nil
}

func (c *Command1) Run(ctx context.Context) error {
	// Invoked when this command is run
	fmt.Println(c.MyURL)
	return nil
}

var cmd1 = command.MustNew(
	"command1",
	"This is command1, a magnificent command that does something.",
	`Longer description...`,
	&Command1{
		MyURL: "default value for --my-url",
	},
	cmd2, // Adding cmd2 as a sub-command of cmd1
)
```

For `root.go`, use this:

```go
package main

import (
	"context"
	"fmt"

	"github.com/arikkfir/command"
)

type Root struct {
	Args []string `args:"true"`
	Port int      `value-name:"PORT" env:"HTTP_PORT"`
}

func (c *Root) PreRun(ctx context.Context) error {
	// Invoked every time this command or any of its sub-commands are run
	fmt.Println(c.Port)
	return nil
}

func (c *Root) Run(ctx context.Context) error {
	// Invoked when this command is run
	fmt.Println(c.Port)
	return nil
}

var root = command.MustNew(
	filepath.Base(os.Args[0]),
	"This is the root command.",
	`This is the command executed when no sub-commands are specified in the command line, e.g. like
running "kubectl" and pressing ENTER.`,
	&Root{
		Port: "default value for --port",
	},
	cmd1, // Adding cmd1 as a sub-command of root
)
```

And finally create `main.go` like so:

```go
package main

import (
	"context"
	"os"

	"github.com/arikkfir/command"
)

func main() {
	command.Execute(context.Context(), os.Stderr, root, os.Args, command.EnvVarsArrayToMap(os.Environ()))
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

type MyCommand struct {
	FlagWithDefaults  string   `flag:"true"`
	ModifyCLIFlagName string   `name:"another-name"`     // Use "another-name" instead of "modify-cli-flag-name"
	ModifyEnvVarName  string   `env:"CUSTOM"`            // Use "CUSTOM" env-var instead of "MODIFY_CLI_ENV_VAR_NAME"
	ModifyValueName   string   `value-name:"PORT"`       // Show "--modify-value-name=PORT" instead of "--modify-value-name=VALUE" on help screen
	ModifyDesc        string   `desc:"Flag description"` // Describe what this flag does
	ModifyRequired    string   `required:"true"`         // Make the flag required
	ModifyInherited   string   `inherited:"true"`        // Sub-commands will get this flag as well
	Args              []string `args:"true"`             // This field will get all the non-flag positional arguments for the command
}
```

## Field types

Configuration fields cab be of type `string`, `int`, `uint`, `float64`, `bool`, or a `struct` containing additional
flags. New types will be added soon (e.g. `time.Time`, `time.Duration`, `net.IP`, and more).

## Contributing

Please do :ok_hand: :muscle: !

See [CONTRIBUTING.md](CONTRIBUTING.md) for more information :pray:

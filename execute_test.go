package command

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	. "github.com/arikkfir/justest"
)

type TrackingAction struct {
	callTime            *time.Time
	errorToReturnOnCall error
}

func (a *TrackingAction) Run(_ context.Context) error {
	a.callTime = ptrOf(time.Now())
	time.Sleep(100 * time.Millisecond)
	return a.errorToReturnOnCall
}

type TrackingPreRunHook struct {
	callTime            *time.Time
	errorToReturnOnCall error
}

func (a *TrackingPreRunHook) PreRun(_ context.Context) error {
	a.callTime = ptrOf(time.Now())
	time.Sleep(100 * time.Millisecond)
	return a.errorToReturnOnCall
}

type ActionWithConfig struct {
	TrackingAction
	MyFlag string `name:"my-flag"`
}

type PreRunHookWithConfig struct {
	TrackingPreRunHook
	MyFlag string `name:"my-flag"`
}

func TestExecute(t *testing.T) {
	t.Parallel()

	t.Run("command must be root", func(t *testing.T) {
		ctx := context.Background()
		child := MustNew("child", "desc", "long desc", nil, nil)
		_ = MustNew("root", "desc", "long desc", nil, nil, child)
		b := &bytes.Buffer{}
		With(t).Verify(Execute(ctx, b, child, nil, nil)).Will(EqualTo(ExitCodeError)).OrFail()
		With(t).Verify(b).Will(Say(`^unsupported operation: command must be the root command$`)).OrFail()
	})

	t.Run("applies configuration", func(t *testing.T) {
		ctx := context.Background()
		cmd := MustNew("cmd", "desc", "long desc", &ActionWithConfig{}, nil)
		With(t).Verify(Execute(ctx, os.Stderr, cmd, []string{"--my-flag=V1"}, nil)).Will(EqualTo(ExitCodeSuccess)).OrFail()
		With(t).Verify(cmd.action.(*ActionWithConfig).MyFlag).Will(EqualTo("V1")).OrFail()
	})

	t.Run("prints usage on CLI parse errors", func(t *testing.T) {
		ctx := context.Background()
		cmd := MustNew("cmd", "desc", "long desc", &ActionWithConfig{}, nil)
		b := &bytes.Buffer{}
		With(t).Verify(Execute(ctx, b, cmd, []string{"--bad-flag=V1"}, nil)).Will(EqualTo(ExitCodeMisconfiguration)).OrFail()
		With(t).Verify(cmd.action.(*ActionWithConfig).MyFlag).Will(BeEmpty()).OrFail()
		With(t).Verify(b.String()).Will(EqualTo("unknown flag: --bad-flag\nUsage: cmd [--help] [--my-flag=VALUE]\n")).OrFail()
	})

	t.Run("prints help on --help flag", func(t *testing.T) {
		ctx := context.Background()
		cmd := MustNew("cmd", "desc", "long desc", &ActionWithConfig{}, nil)
		b := &bytes.Buffer{}
		With(t).Verify(Execute(ctx, b, cmd, []string{"--help"}, nil)).Will(EqualTo(ExitCodeSuccess)).OrFail()
		With(t).Verify(b.String()).Will(EqualTo(`
cmd: desc

Description: long desc

Usage:
    cmd [--help] [--my-flag=VALUE]

Flags:
    [--help]            Show this help screen and exit. (default value: false, 
                        environment variable: HELP)
    [--my-flag=VALUE]   environment variable: MY_FLAG

`[1:])).OrFail()
	})

	t.Run("preRun called for command chain", func(t *testing.T) {
		ctx := context.Background()
		sub2 := MustNew("sub2", "desc", "long desc", &ActionWithConfig{}, []PreRunHook{&PreRunHookWithConfig{}})
		sub1 := MustNew("sub1", "desc", "long desc", &ActionWithConfig{}, []PreRunHook{&PreRunHookWithConfig{}}, sub2)
		root := MustNew("cmd", "desc", "long desc", &ActionWithConfig{}, []PreRunHook{&PreRunHookWithConfig{}}, sub1)
		With(t).Verify(Execute(ctx, os.Stderr, root, []string{"sub1", "sub2"}, nil)).Will(EqualTo(ExitCodeSuccess)).OrFail()

		sub2PreRunTime := sub2.preRunHooks[0].(*PreRunHookWithConfig).callTime
		With(t).Verify(sub2PreRunTime).Will(Not(BeNil())).OrFail()

		sub1PreRunTime := sub1.preRunHooks[0].(*PreRunHookWithConfig).callTime
		With(t).Verify(sub1PreRunTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub1PreRunTime.Before(*sub2PreRunTime)).Will(EqualTo(true)).OrFail()

		rootPreRunTime := root.preRunHooks[0].(*PreRunHookWithConfig).callTime
		With(t).Verify(rootPreRunTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(rootPreRunTime.Before(*sub1PreRunTime)).Will(EqualTo(true)).OrFail()

		sub2RunTime := sub2.action.(*ActionWithConfig).callTime
		With(t).Verify(sub2RunTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub2RunTime.After(*sub2PreRunTime)).Will(EqualTo(true)).OrFail()
	})
}

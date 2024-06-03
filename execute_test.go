package command

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	. "github.com/arikkfir/justest"
)

type TrackingExecutor struct {
	preRunCalled        *time.Time
	preRunErrorToReturn error
	runCalled           *time.Time
	runErrorToReturn    error
}

func (te *TrackingExecutor) PreRun(_ context.Context) error {
	te.preRunCalled = ptrOf(time.Now())
	time.Sleep(100 * time.Millisecond)
	return te.preRunErrorToReturn
}

func (te *TrackingExecutor) Run(_ context.Context) error {
	te.runCalled = ptrOf(time.Now())
	time.Sleep(100 * time.Millisecond)
	return te.runErrorToReturn
}

type ExecutorWithFlag struct {
	MyFlag string `name:"my-flag"`
}

func (e *ExecutorWithFlag) PreRun(_ context.Context) error {
	return nil
}

func (e *ExecutorWithFlag) Run(_ context.Context) error {
	return nil
}

func TestExecute(t *testing.T) {
	t.Parallel()

	t.Run("command must be root", func(t *testing.T) {
		ctx := context.Background()
		child := MustNew("child", "desc", "long desc", InlineExecutor{})
		_ = MustNew("root", "desc", "long desc", InlineExecutor{}, child)
		b := &bytes.Buffer{}
		With(t).Verify(Execute(ctx, b, child, nil, nil)).Will(EqualTo(ExitCodeError)).OrFail()
		With(t).Verify(b).Will(Say(`^unsupported operation: command must be the root command$`)).OrFail()
	})

	t.Run("applies configuration", func(t *testing.T) {
		ctx := context.Background()
		cmd := MustNew("cmd", "desc", "long desc", &ExecutorWithFlag{})
		With(t).Verify(Execute(ctx, os.Stderr, cmd, []string{"--my-flag=V1"}, nil)).Will(EqualTo(ExitCodeSuccess)).OrFail()
		With(t).Verify(cmd.executor.(*ExecutorWithFlag).MyFlag).Will(EqualTo("V1")).OrFail()
	})

	t.Run("prints usage on CLI parse errors", func(t *testing.T) {
		ctx := context.Background()
		cmd := MustNew("cmd", "desc", "long desc", &ExecutorWithFlag{})
		b := &bytes.Buffer{}
		With(t).Verify(Execute(ctx, b, cmd, []string{"--bad-flag=V1"}, nil)).Will(EqualTo(ExitCodeMisconfiguration)).OrFail()
		With(t).Verify(cmd.executor.(*ExecutorWithFlag).MyFlag).Will(BeEmpty()).OrFail()
		With(t).Verify(b.String()).Will(EqualTo("unknown flag: --bad-flag\nUsage: cmd [--help] [--my-flag=VALUE]\n")).OrFail()
	})

	t.Run("prints help on --help flag", func(t *testing.T) {
		ctx := context.Background()
		cmd := MustNew("cmd", "desc", "long desc", &ExecutorWithFlag{})
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
		sub2 := MustNew("sub2", "desc", "long desc", &TrackingExecutor{})
		sub1 := MustNew("sub1", "desc", "long desc", &TrackingExecutor{}, sub2)
		root := MustNew("cmd", "desc", "long desc", &TrackingExecutor{}, sub1)
		With(t).Verify(Execute(ctx, os.Stderr, root, []string{"sub1", "sub2"}, nil)).Will(EqualTo(ExitCodeSuccess)).OrFail()

		sub2PreRunTime := sub2.executor.(*TrackingExecutor).preRunCalled
		With(t).Verify(sub2PreRunTime).Will(Not(BeNil())).OrFail()

		sub1PreRunTime := sub1.executor.(*TrackingExecutor).preRunCalled
		With(t).Verify(sub1PreRunTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub1PreRunTime.Before(*sub2PreRunTime)).Will(EqualTo(true)).OrFail()

		rootPreRunTime := root.executor.(*TrackingExecutor).preRunCalled
		With(t).Verify(rootPreRunTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(rootPreRunTime.Before(*sub1PreRunTime)).Will(EqualTo(true)).OrFail()

		sub2RunTime := sub2.executor.(*TrackingExecutor).runCalled
		With(t).Verify(sub2RunTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub2RunTime.After(*sub2PreRunTime)).Will(EqualTo(true)).OrFail()
	})
}

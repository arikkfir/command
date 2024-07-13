package command

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/arikkfir/justest"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type TrackingAction struct {
	callTime            *time.Time
	providedCtx         context.Context
	errorToReturnOnCall error
}

func (a *TrackingAction) Run(ctx context.Context) error {
	a.callTime = ptrOf(time.Now())
	a.providedCtx = ctx
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

type TrackingPostRunHook struct {
	callTime            *time.Time
	providedCtx         context.Context
	providedActionError error
	providedExitCode    ExitCode
	errorToReturnOnCall error
}

func (a *TrackingPostRunHook) PostRun(ctx context.Context, actionError error, exitCode ExitCode) error {
	a.callTime = ptrOf(time.Now())
	a.providedCtx = ctx
	a.providedActionError = actionError
	a.providedExitCode = exitCode
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

type PostRunHookWithConfig struct {
	TrackingPostRunHook
	MyFlag string `name:"my-flag"`
}

func TestExecute(t *testing.T) {
	t.Parallel()

	t.Run("command must be root", func(t *testing.T) {
		ctx := context.Background()
		child := MustNew("child", "desc", "long desc", nil, nil)
		_ = MustNew("root", "desc", "long desc", nil, nil, child)
		b := &bytes.Buffer{}
		With(t).Verify(ExecuteWithContext(ctx, b, child, nil, nil)).Will(EqualTo(ExitCodeError)).OrFail()
		With(t).Verify(b).Will(Say(`^unsupported operation: command must be the root command$`)).OrFail()
	})

	t.Run("applies configuration", func(t *testing.T) {
		ctx := context.Background()
		cmd := MustNew("cmd", "desc", "long desc", &ActionWithConfig{}, nil)
		With(t).Verify(ExecuteWithContext(ctx, os.Stderr, cmd, []string{"--my-flag=V1"}, nil)).Will(EqualTo(ExitCodeSuccess)).OrFail()
		With(t).Verify(cmd.action.(*ActionWithConfig).MyFlag).Will(EqualTo("V1")).OrFail()
	})

	t.Run("prints usage on CLI parse errors", func(t *testing.T) {
		ctx := context.Background()
		cmd := MustNew("cmd", "desc", "long desc", &ActionWithConfig{}, nil)
		b := &bytes.Buffer{}
		With(t).Verify(ExecuteWithContext(ctx, b, cmd, []string{"--bad-flag=V1"}, nil)).Will(EqualTo(ExitCodeMisconfiguration)).OrFail()
		With(t).Verify(cmd.action.(*ActionWithConfig).MyFlag).Will(BeEmpty()).OrFail()
		With(t).Verify(b.String()).Will(EqualTo("unknown flag: --bad-flag\nUsage: cmd [--help] [--my-flag=VALUE]\n")).OrFail()
	})

	t.Run("prints help on --help flag", func(t *testing.T) {
		ctx := context.Background()
		cmd := MustNew("cmd", "desc", "long desc", &ActionWithConfig{}, nil)
		b := &bytes.Buffer{}
		With(t).Verify(ExecuteWithContext(ctx, b, cmd, []string{"--help"}, nil)).Will(EqualTo(ExitCodeSuccess)).OrFail()
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
		sub2 := MustNew("sub2", "desc", "long desc", &ActionWithConfig{}, []any{&PreRunHookWithConfig{}})
		sub1 := MustNew("sub1", "desc", "long desc", nil, []any{&PreRunHookWithConfig{}}, sub2)
		root := MustNew("cmd", "desc", "long desc", nil, []any{&PreRunHookWithConfig{}}, sub1)
		With(t).Verify(ExecuteWithContext(ctx, os.Stderr, root, []string{"sub1", "sub2"}, nil)).Will(EqualTo(ExitCodeSuccess)).OrFail()

		rootPreRunHook := root.preRunHooks[0].(*PreRunHookWithConfig)
		sub1PreRunHook := sub1.preRunHooks[0].(*PreRunHookWithConfig)
		sub2PreRunHook := sub2.preRunHooks[0].(*PreRunHookWithConfig)
		sub2Action := sub2.action.(*ActionWithConfig)

		With(t).Verify(rootPreRunHook.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(rootPreRunHook.callTime.Before(*sub1PreRunHook.callTime)).Will(EqualTo(true)).OrFail()
		With(t).Verify(sub1PreRunHook.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub1PreRunHook.callTime.Before(*sub2PreRunHook.callTime)).Will(EqualTo(true)).OrFail()
		With(t).Verify(sub2PreRunHook.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub2PreRunHook.callTime.Before(*sub2Action.callTime)).Will(EqualTo(true)).OrFail()
		With(t).Verify(sub2Action.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub2Action.callTime.After(*sub2PreRunHook.callTime)).Will(EqualTo(true)).OrFail()
	})

	t.Run("preRun failure stops execution", func(t *testing.T) {
		failingPreHook := &PreRunHookWithConfig{TrackingPreRunHook: TrackingPreRunHook{errorToReturnOnCall: fmt.Errorf("fail")}}
		passThroughPreHook := func() PreRunHook { return &PreRunHookWithConfig{} }

		ctx := context.Background()
		sub2 := MustNew("sub2", "desc", "long desc", &ActionWithConfig{}, []any{passThroughPreHook()})
		sub1 := MustNew("sub1", "desc", "long desc", nil, []any{passThroughPreHook(), failingPreHook}, sub2)
		root := MustNew("cmd", "desc", "long desc", nil, []any{passThroughPreHook()}, sub1)

		rootPreRunHook := root.preRunHooks[0].(*PreRunHookWithConfig)
		sub1PreRunHook := sub1.preRunHooks[0].(*PreRunHookWithConfig)
		sub2PreRunHook := sub2.preRunHooks[0].(*PreRunHookWithConfig)
		sub2Action := sub2.action.(*ActionWithConfig)

		With(t).Verify(ExecuteWithContext(ctx, os.Stderr, root, []string{"sub1", "sub2"}, nil)).Will(EqualTo(ExitCodeError)).OrFail()
		With(t).Verify(rootPreRunHook.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(rootPreRunHook.callTime.Before(*sub1PreRunHook.callTime)).Will(EqualTo(true)).OrFail()
		With(t).Verify(sub1PreRunHook.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub2PreRunHook.callTime).Will(BeNil()).OrFail()
		With(t).Verify(sub2Action.callTime).Will(BeNil()).OrFail()
	})

	t.Run("postRun called for command chain", func(t *testing.T) {
		ctx := context.Background()
		sub2 := MustNew("sub2", "desc", "long desc", &ActionWithConfig{}, []any{&PostRunHookWithConfig{}})
		sub1 := MustNew("sub1", "desc", "long desc", nil, []any{&PostRunHookWithConfig{}}, sub2)
		root := MustNew("cmd", "desc", "long desc", nil, []any{&PostRunHookWithConfig{}}, sub1)

		exitCode := ExecuteWithContext(ctx, os.Stderr, root, []string{"sub1", "sub2"}, nil)
		With(t).Verify(exitCode).Will(EqualTo(ExitCodeSuccess)).OrFail()

		rootPostRunHook := root.postRunHooks[0].(*PostRunHookWithConfig)
		sub1PostRunHook := sub1.postRunHooks[0].(*PostRunHookWithConfig)
		sub2PostRunHook := sub2.postRunHooks[0].(*PostRunHookWithConfig)
		sub2Action := sub2.action.(*ActionWithConfig)

		With(t).Verify(sub2Action.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub2Action.callTime.Before(*sub2PostRunHook.callTime)).Will(EqualTo(true)).OrFail()
		With(t).Verify(sub2PostRunHook.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub2PostRunHook.callTime.Before(*sub1PostRunHook.callTime)).Will(EqualTo(true)).OrFail()
		With(t).Verify(sub2PostRunHook.providedActionError).Will(EqualTo(sub2Action.errorToReturnOnCall)).OrFail()
		With(t).Verify(sub2PostRunHook.providedExitCode).Will(EqualTo(exitCode)).OrFail()
		With(t).Verify(sub1PostRunHook.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub1PostRunHook.callTime.Before(*rootPostRunHook.callTime)).Will(EqualTo(true)).OrFail()
		With(t).Verify(sub1PostRunHook.providedActionError).Will(EqualTo(sub2PostRunHook.errorToReturnOnCall)).OrFail()
		With(t).Verify(sub1PostRunHook.providedExitCode).Will(EqualTo(exitCode)).OrFail()
		With(t).Verify(rootPostRunHook.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(rootPostRunHook.providedActionError).Will(BeNil()).OrFail()
		With(t).Verify(rootPostRunHook.providedExitCode).Will(EqualTo(exitCode)).OrFail()
	})

	t.Run("postRun chain called in full, even on action or hook error", func(t *testing.T) {
		failingPostHook := func() PostRunHook {
			return &PostRunHookWithConfig{TrackingPostRunHook: TrackingPostRunHook{errorToReturnOnCall: fmt.Errorf("failing post hook")}}
		}
		passThroughPostHook := func() PostRunHook { return &PostRunHookWithConfig{} }
		failingAction := &ActionWithConfig{TrackingAction: TrackingAction{errorToReturnOnCall: fmt.Errorf("failing action")}}

		ctx := context.Background()
		sub2 := MustNew("sub2", "desc", "long desc", failingAction, []any{failingPostHook()})
		sub1 := MustNew("sub1", "desc", "long desc", nil, []any{passThroughPostHook()}, sub2)
		root := MustNew("cmd", "desc", "long desc", nil, []any{passThroughPostHook()}, sub1)

		exitCode := ExecuteWithContext(ctx, os.Stderr, root, []string{"sub1", "sub2"}, nil)
		With(t).Verify(exitCode).Will(EqualTo(ExitCodeError)).OrFail()

		rootPostRunHook := root.postRunHooks[0].(*PostRunHookWithConfig)
		sub1PostRunHook := sub1.postRunHooks[0].(*PostRunHookWithConfig)
		sub2PostRunHook := sub2.postRunHooks[0].(*PostRunHookWithConfig)
		sub2Action := sub2.action.(*ActionWithConfig)

		With(t).Verify(sub2Action.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub2Action.callTime.Before(*sub2PostRunHook.callTime)).Will(EqualTo(true)).OrFail()
		With(t).Verify(sub2PostRunHook.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub2PostRunHook.callTime.Before(*sub1PostRunHook.callTime)).Will(EqualTo(true)).OrFail()
		With(t).Verify(sub2PostRunHook.providedActionError).Will(EqualTo(sub2Action.errorToReturnOnCall, cmpopts.EquateErrors())).OrFail()
		With(t).Verify(sub2PostRunHook.providedExitCode).Will(EqualTo(exitCode)).OrFail()
		With(t).Verify(sub1PostRunHook.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(sub1PostRunHook.callTime.Before(*rootPostRunHook.callTime)).Will(EqualTo(true)).OrFail()
		With(t).Verify(sub1PostRunHook.providedActionError).Will(EqualTo(sub2Action.errorToReturnOnCall, cmpopts.EquateErrors())).OrFail()
		With(t).Verify(sub1PostRunHook.providedExitCode).Will(EqualTo(exitCode)).OrFail()
		With(t).Verify(rootPostRunHook.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(rootPostRunHook.providedActionError).Will(EqualTo(sub2Action.errorToReturnOnCall, cmpopts.EquateErrors())).OrFail()
		With(t).Verify(rootPostRunHook.providedExitCode).Will(EqualTo(exitCode)).OrFail()
	})

	t.Run("missing required flags fail execution", func(t *testing.T) {
		type ActionWithRequiredFlag struct {
			TrackingAction
			MyFlag string `required:"true"`
		}
		action := &ActionWithRequiredFlag{}
		ctx := context.Background()
		root := MustNew("cmd", "desc", "long desc", action, nil)

		b := &bytes.Buffer{}
		With(t).Verify(ExecuteWithContext(ctx, b, root, nil, nil)).Will(EqualTo(ExitCodeMisconfiguration)).OrFail()
		With(t).Verify(action.TrackingAction.callTime).Will(BeNil()).OrFail()
		With(t).Verify(b.String()).Will(EqualTo("required flag is missing: --my-flag\nUsage: cmd [--help] --my-flag=VALUE\n")).OrFail()
	})

	t.Run("required flags with default value do not fail execution", func(t *testing.T) {
		type ActionWithRequiredFlag struct {
			TrackingAction
			MyFlag string `required:"true"`
		}
		action := &ActionWithRequiredFlag{
			MyFlag: "abc",
		}
		ctx := context.Background()
		root := MustNew("cmd", "desc", "long desc", action, nil)

		b := &bytes.Buffer{}
		With(t).Verify(ExecuteWithContext(ctx, b, root, nil, nil)).Will(EqualTo(ExitCodeSuccess)).OrFail()
		With(t).Verify(action.TrackingAction.callTime).Will(Not(BeNil())).OrFail()
		With(t).Verify(b.String()).Will(BeEmpty()).OrFail()
	})

	t.Run("ensure post-hooks use fresh context", func(t *testing.T) {
		//nolint:all
		executionCtx := context.WithValue(context.Background(), "k", "v")

		action := &TrackingAction{}
		root := MustNew("cmd", "desc", "long desc", action, []any{&PostRunHookWithConfig{}})

		exitCode := ExecuteWithContext(executionCtx, os.Stderr, root, nil, nil)
		With(t).Verify(exitCode).Will(EqualTo(ExitCodeSuccess)).OrFail()

		if action.providedCtx != executionCtx {
			t.Fatalf("incorrect context passed to action: %+v", action.providedCtx)
		}

		rootPostRunHook := root.postRunHooks[0].(*PostRunHookWithConfig)
		if rootPostRunHook.providedCtx == executionCtx {
			t.Fatalf("incorrect context passed to posthook: %+v", rootPostRunHook.providedCtx)
		}
	})
}

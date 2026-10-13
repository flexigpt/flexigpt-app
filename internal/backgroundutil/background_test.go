package backgroundutil

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRunSafelyReportsTypedFailureWithoutPanicValue(t *testing.T) {
	const secret = "private-provider-credential"

	var reported *PanicFailureError
	RunSafely(
		"provider execution",
		func() {
			panic(secret)
		},
		func(failure PanicFailureError) {
			reported = &failure
		},
	)

	if reported == nil {
		t.Fatal("panic was not reported")
	}
	if reported.Operation != "provider execution" {
		t.Fatalf("unexpected operation: %q", reported.Operation)
	}
	if len(reported.Stack) == 0 {
		t.Fatal("panic stack was not captured")
	}
	if strings.Contains(reported.Error(), secret) {
		t.Fatal("public failure contains the panic value")
	}
}

func TestLaunchWaitsForPanicHandler(t *testing.T) {
	var group sync.WaitGroup
	var handled bool

	Launch(
		&group,
		"background execution",
		func() {
			panic("execution failed")
		},
		func(PanicFailureError) {
			handled = true
		},
	)

	group.Wait()
	if !handled {
		t.Fatal("Wait returned before the panic handler completed")
	}
}

func TestLaunchContainsPanicHandlerFailure(t *testing.T) {
	var group sync.WaitGroup

	Launch(
		&group,
		"background execution",
		func() {
			panic("execution failed")
		},
		func(PanicFailureError) {
			panic("failure handler failed")
		},
	)

	group.Wait()
}

func TestAfterFuncContainsPanic(t *testing.T) {
	reported := make(chan PanicFailureError, 1)

	timer := AfterFunc(
		0,
		"timer execution",
		func() {
			panic("timer failed")
		},
		func(failure PanicFailureError) {
			reported <- failure
		},
	)
	defer timer.Stop()

	select {
	case failure := <-reported:
		if failure.Operation != "timer execution" {
			t.Fatalf("unexpected operation: %q", failure.Operation)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timer panic was not reported")
	}
}

func TestAfterContextContainsPanic(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	reported := make(chan PanicFailureError, 1)
	stop := AfterContext(
		ctx,
		"context cleanup",
		func() {
			panic("cleanup failed")
		},
		func(failure PanicFailureError) {
			reported <- failure
		},
	)
	defer stop()

	cancel()

	select {
	case failure := <-reported:
		if failure.Operation != "context cleanup" {
			t.Fatalf("unexpected operation: %q", failure.Operation)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("context callback panic was not reported")
	}
}

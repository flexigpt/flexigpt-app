package jsonutil

import (
	"strings"
	"sync"
	"testing"
)

func TestCanonicalizeProducesStableJSONAndNumbers(t *testing.T) {
	t.Parallel()

	canonical, err := Canonicalize([]byte(`{"b":1.00,"a":[true,1e2,-0,0.0100]}`))
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	const want = `{"a":[true,1e2,0,1e-2],"b":1}`
	if string(canonical) != want {
		t.Fatalf("canonical=%s, want %s", canonical, want)
	}

	empty, err := CanonicalizeObject(nil, len(EmptyObject))
	if err != nil {
		t.Fatalf("CanonicalizeObject(nil): %v", err)
	}
	if string(empty) != EmptyObject {
		t.Fatalf("empty canonical object=%q, want %q", empty, EmptyObject)
	}
	if !Equal([]byte(`{"a":1,"b":[2,3]}`), []byte(` { "b" : [2.0,3e0], "a":1e0 } `)) {
		t.Fatal("Equal did not recognize semantically equivalent JSON")
	}
}

func TestCanonicalizeRejectsMalformedAndUnsafeInput(t *testing.T) {
	t.Parallel()

	deep := strings.Repeat(`{"x":`, maximumDepth+1) + "0" + strings.Repeat("}", maximumDepth+1)
	invalid := [][]byte{
		[]byte(`{"a":1,"a":2}`),
		[]byte(`{"a":1} {"b":2}`),
		[]byte(`{"a":01}`),
		[]byte(deep),
		{0xff},
	}
	for _, raw := range invalid {
		t.Run(string(raw), func(t *testing.T) {
			t.Parallel()
			if _, err := Canonicalize(raw); err == nil {
				t.Fatalf("Canonicalize(%q) succeeded", raw)
			}
		})
	}

	array, err := Canonicalize([]byte(`[1,2,3]`))
	if err != nil || string(array) != `[1,2,3]` {
		t.Fatalf("Canonicalize(array)=%q err=%v", array, err)
	}
	if _, err := CanonicalizeObject([]byte(`[]`), 16); err == nil {
		t.Fatal("CanonicalizeObject(array) succeeded")
	}
	if _, err := CanonicalizeObject([]byte(`{"a":1}`), 2); err == nil {
		t.Fatal("CanonicalizeObject oversize input succeeded")
	}
}

func TestCanonicalizeIsDeterministicForConcurrentCallers(t *testing.T) {
	t.Parallel()

	const (
		workers = 48
		input   = `{"z":{"b":2,"a":1},"n":1.2300e+2}`
		want    = `{"n":1.23e2,"z":{"a":1,"b":2}}`
	)
	var group sync.WaitGroup
	errorsSeen := make(chan error, workers)
	for range workers {
		group.Go(func() {
			value, err := Canonicalize([]byte(input))
			if err != nil {
				errorsSeen <- err
				return
			}
			if string(value) != want {
				errorsSeen <- &canonicalizationMismatchError{got: string(value), want: want}
			}
		})
	}
	group.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Fatalf("concurrent canonicalization failed: %v", err)
	}
}

type canonicalizationMismatchError struct {
	got  string
	want string
}

func (e *canonicalizationMismatchError) Error() string {
	return "canonical JSON mismatch: got " + e.got + ", want " + e.want
}

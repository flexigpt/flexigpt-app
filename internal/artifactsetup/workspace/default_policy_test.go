package workspace

import (
	"bytes"
	"io/fs"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/provider/iofs"
)

func TestDefaultPolicyProviderRegistrationsUseFlatEmbeddedPolicy(
	t *testing.T,
) {
	t.Parallel()

	source, err := defaultPolicySource()
	if err != nil {
		t.Fatalf("load default Workspace policy source: %v", err)
	}
	if string(source.Root) != defaultPolicyRoot {
		t.Fatalf(
			"default Workspace policy root = %q, want %q",
			source.Root,
			defaultPolicyRoot,
		)
	}
	if string(source.Locator) != defaultPolicyLocator {
		t.Fatalf(
			"default Workspace policy locator = %q, want %q",
			source.Locator,
			defaultPolicyLocator,
		)
	}

	registrations, err := DefaultPolicyProviderRegistrations(t.Context())
	if err != nil {
		t.Fatalf(
			"create default Workspace policy provider registrations: %v",
			err,
		)
	}
	if len(registrations) != 1 {
		t.Fatalf(
			"default Workspace policy registrations = %d, want 1",
			len(registrations),
		)
	}

	registration, found := registrations[source.ProviderKey]
	if !found {
		t.Fatalf(
			"default Workspace policy registration %q is absent",
			source.ProviderKey,
		)
	}
	if registration.Filesystem == nil {
		t.Fatal("default Workspace policy registration filesystem is nil")
	}

	raw, err := fs.ReadFile(
		registration.Filesystem,
		defaultPolicyLocator,
	)
	if err != nil {
		t.Fatalf(
			"read default Workspace policy %q: %v",
			defaultPolicyLocator,
			err,
		)
	}
	if !bytes.Equal(raw, source.Policy.RawYAML) {
		t.Fatalf(
			"embedded Workspace policy %q differs from source raw YAML",
			defaultPolicyLocator,
		)
	}

	if registration.Immutable == nil {
		t.Fatal("default Workspace policy immutable evidence is nil")
	}
	if registration.Immutable.Revision != source.Policy.Version {
		t.Fatalf(
			"default Workspace policy revision = %q, want %q",
			registration.Immutable.Revision,
			source.Policy.Version,
		)
	}

	actualDigest, err := iofs.ContentDigest(
		t.Context(),
		registration.Filesystem,
	)
	if err != nil {
		t.Fatalf("digest default Workspace policy filesystem: %v", err)
	}
	if registration.Immutable.ContentDigest != actualDigest {
		t.Fatalf(
			"default Workspace policy digest = %q, want %q",
			registration.Immutable.ContentDigest,
			actualDigest,
		)
	}
}

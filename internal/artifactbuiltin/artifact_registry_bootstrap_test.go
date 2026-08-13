package artifactbuiltin

import (
	"context"
	"slices"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/topology"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

type bootstrapFinalizationTestEnsurer struct {
	trace *[]string
}

func (e bootstrapFinalizationTestEnsurer) EnsureProtectedTopology(
	_ context.Context,
	_ topology.Declaration,
) (topology.Installed, error) {
	*e.trace = append(*e.trace, "topology")
	return topology.Installed{}, nil
}

type bootstrapFinalizationTestHydrator struct {
	trace *[]string
}

func (h bootstrapFinalizationTestHydrator) PrepareTopologyHydrations(
	_ context.Context,
	desired []topology.Hydration,
) (map[string]bool, error) {
	*h.trace = append(*h.trace, "prepare")

	current := make(map[string]bool, len(desired))
	for _, value := range desired {
		current[value.InstallerName] = false
	}
	return current, nil
}

func (h bootstrapFinalizationTestHydrator) CommitTopologyHydration(
	_ context.Context,
	desired topology.Hydration,
) error {
	*h.trace = append(*h.trace, "commit:"+desired.InstallerName)
	return nil
}

type bootstrapFinalizationTestInstaller struct {
	name      string
	hydration topology.Hydration
	trace     *[]string
}

func (i bootstrapFinalizationTestInstaller) BuiltInName() string {
	return i.name
}

func (bootstrapFinalizationTestInstaller) BuiltInIDs() []string {
	return nil
}

func (bootstrapFinalizationTestInstaller) BuiltInPackageScopes() []basespec.Locator {
	return nil
}

func (i bootstrapFinalizationTestInstaller) Ensure(context.Context) error {
	*i.trace = append(*i.trace, "ensure:"+i.name)
	return nil
}

func (i bootstrapFinalizationTestInstaller) DesiredHydration(
	context.Context,
) (topology.Hydration, error) {
	return i.hydration, nil
}

func (i bootstrapFinalizationTestInstaller) EnsureHydration(
	context.Context,
	bool,
) error {
	*i.trace = append(*i.trace, "ensure:"+i.name)
	return nil
}

func (i bootstrapFinalizationTestInstaller) FinalizeHydration(
	context.Context,
) error {
	*i.trace = append(*i.trace, "finalize:"+i.name)
	return nil
}

func TestBootstrapFinalizesHydrationAfterAllInstallersPublish(
	t *testing.T,
) {
	t.Parallel()

	declaration := BuiltinTopologyDeclaration()
	trace := make([]string, 0)

	registry, err := NewBootstrapRegistry(
		declaration,
		bootstrapFinalizationTestEnsurer{trace: &trace},
		bootstrapFinalizationTestHydrator{trace: &trace},
	)
	if err != nil {
		t.Fatalf("NewBootstrapRegistry: %v", err)
	}

	for _, name := range []string{"mcp.bundle", "agent.skill"} {
		installer := bootstrapFinalizationTestInstaller{
			name: name,
			hydration: topology.Hydration{
				InstallerName: name,
				RootID:        declaration.Root.ID,
				SourceID:      declaration.Sources[0].ID,
				Fingerprint:   cryptoutil.DigestBytes([]byte(name)),
			},
			trace: &trace,
		}
		if err := registry.Register(installer); err != nil {
			t.Fatalf("Register(%q): %v", name, err)
		}
	}

	if err := registry.Ensure(t.Context()); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	want := []string{
		"prepare",
		"topology",
		"ensure:agent.skill",
		"ensure:mcp.bundle",
		"finalize:agent.skill",
		"finalize:mcp.bundle",
		"commit:agent.skill",
		"commit:mcp.bundle",
	}
	if !slices.Equal(trace, want) {
		t.Fatalf("bootstrap trace=%#v, want %#v", trace, want)
	}
}

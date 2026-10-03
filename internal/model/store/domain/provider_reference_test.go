package domain

import (
	"slices"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	rootModel "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/root/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/spec"
)

func TestArtifactNameReferenceLookupRoots(t *testing.T) {
	const (
		userRoot    rootModel.RootID = "0192c4c0-0000-7000-8000-000000000002"
		builtinRoot rootModel.RootID = "0192c4c0-0000-7000-8000-000000000001"
	)

	tests := []struct {
		name      string
		reference declaration.ArtifactNameReference
		current   rootModel.RootID
		builtin   rootModel.RootID
		want      []rootModel.RootID
		wantErr   bool
	}{
		{
			name: "unscoped searches current then builtin",
			reference: declaration.ArtifactNameReference{
				Name: spec.LogicalName("provider"),
			},
			current: userRoot,
			builtin: builtinRoot,
			want:    []rootModel.RootID{userRoot, builtinRoot},
		},
		{
			name: "builtin scope searches only builtin",
			reference: declaration.ArtifactNameReference{
				Name:  spec.LogicalName("provider"),
				Scope: declaration.LookupScopeBuiltin,
			},
			current: userRoot,
			builtin: builtinRoot,
			want:    []rootModel.RootID{builtinRoot},
		},
		{
			name: "current builtin root is not repeated",
			reference: declaration.ArtifactNameReference{
				Name: spec.LogicalName("provider"),
			},
			current: builtinRoot,
			builtin: builtinRoot,
			want:    []rootModel.RootID{builtinRoot},
		},
		{
			name: "builtin scope requires configured builtin root",
			reference: declaration.ArtifactNameReference{
				Name:  spec.LogicalName("provider"),
				Scope: declaration.LookupScopeBuiltin,
			},
			current: userRoot,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ArtifactNameReferenceLookupRoots(
				test.reference,
				test.current,
				test.builtin,
			)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ArtifactNameReferenceLookupRoots: %v", err)
			}
			if !slices.Equal(got, test.want) {
				t.Fatalf("roots got %#v, want %#v", got, test.want)
			}
		})
	}
}

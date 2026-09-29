package domain

import (
	"slices"
	"testing"

	"github.com/flexigpt/flexigpt-app/internal/artifactcontract/declaration"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
)

func TestArtifactNameReferenceLookupRoots(t *testing.T) {
	const (
		userRoot    root.RootID = "0192c4c0-0000-7000-8000-000000000002"
		builtinRoot root.RootID = "0192c4c0-0000-7000-8000-000000000001"
	)

	tests := []struct {
		name      string
		reference declaration.ArtifactNameReference
		current   root.RootID
		builtin   root.RootID
		want      []root.RootID
		wantErr   bool
	}{
		{
			name: "unscoped searches current then builtin",
			reference: declaration.ArtifactNameReference{
				Name: basespec.LogicalName("provider"),
			},
			current: userRoot,
			builtin: builtinRoot,
			want:    []root.RootID{userRoot, builtinRoot},
		},
		{
			name: "builtin scope searches only builtin",
			reference: declaration.ArtifactNameReference{
				Name:  basespec.LogicalName("provider"),
				Scope: declaration.LookupScopeBuiltin,
			},
			current: userRoot,
			builtin: builtinRoot,
			want:    []root.RootID{builtinRoot},
		},
		{
			name: "current builtin root is not repeated",
			reference: declaration.ArtifactNameReference{
				Name: basespec.LogicalName("provider"),
			},
			current: builtinRoot,
			builtin: builtinRoot,
			want:    []root.RootID{builtinRoot},
		},
		{
			name: "builtin scope requires configured builtin root",
			reference: declaration.ArtifactNameReference{
				Name:  basespec.LogicalName("provider"),
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

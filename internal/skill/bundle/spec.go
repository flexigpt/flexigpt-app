package bundle

import (
	agentskillsSpec "github.com/flexigpt/agentskills-go/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/artifact"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/catalog"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/collection"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/source"
	"github.com/flexigpt/flexigpt-app/internal/cryptoutil"
)

const (
	DiscoveryPolicyRevision = "skill.bundle.discovery.v1"

	RoleManaged  basespec.AttachmentRole = "managed"
	RoleBuiltIn  basespec.AttachmentRole = "builtin"
	RoleExternal basespec.AttachmentRole = "external"
	RoleLibrary  basespec.AttachmentRole = "library"
)

type AttachmentDraft struct {
	SourceID              basespec.SourceID
	Role                  basespec.AttachmentRole
	Enabled               bool
	DiscoveryRoot         basespec.Locator
	ExpectedMemberDigests map[basespec.Locator]cryptoutil.Digest
}
type CreateBundleRequest struct {
	RootID                  basespec.RootID
	CollectionID            basespec.CollectionID
	ManagedSourceID         basespec.SourceID
	ManagedSourceStorageKey basespec.StorageKey
	DisplayName             string
	Description             string
	Enabled                 bool
	LogicalName             basespec.LogicalName
	LogicalVersion          basespec.LogicalVersion
	Labels                  map[string]string
	Attachments             []AttachmentDraft
}

type UpdateBundleRequest struct {
	Bundle           collection.CollectionRef
	ExpectedRevision uint64
	DisplayName      string
	Description      string
	Enabled          bool
}

type CreateManagedSkillRequest struct {
	Bundle                     collection.CollectionRef
	ExpectedCollectionRevision uint64
	ArtifactID                 basespec.ArtifactID
	SkillName                  string
	SKILLMD                    []byte
	ExpectedArtifactRevision   uint64

	// Document is an optional structured authoring input. Serialization is
	// delegated to agentskills-go. It is mutually exclusive with SKILLMD.
	Document *agentskillsSpec.SkillDocument
	Files    []source.ManagedPackageFile
	Enabled  bool
}

type CreateManagedSkillResponse struct {
	Artifact artifact.Artifact
	Address  artifact.ArtifactAddress
}

type AdoptSkillRequest struct {
	Bundle                  collection.CollectionRef
	Occurrence              catalog.OccurrenceKey
	ArtifactID              basespec.ArtifactID
	ExpectedCatalogRevision uint64
	Name                    string
	Enabled                 bool
}

type PinSkillRequest struct {
	Bundle                     collection.CollectionRef
	ExpectedCollectionRevision uint64
	ArtifactID                 basespec.ArtifactID
	Binding                    artifact.SourceBinding
	Name                       string
	Enabled                    bool
}

type BuiltInBundleTopology struct {
	RootID                basespec.RootID                        `json:"-"`
	CollectionID          basespec.CollectionID                  `json:"-"`
	SourceID              basespec.SourceID                      `json:"-"`
	LogicalName           basespec.LogicalName                   `json:"-"`
	LogicalVersion        basespec.LogicalVersion                `json:"-"`
	DisplayName           string                                 `json:"-"`
	Description           string                                 `json:"-"`
	Labels                map[string]string                      `json:"-"`
	Enabled               bool                                   `json:"-"`
	DiscoveryRoot         basespec.Locator                       `json:"-"`
	ExpectedMemberDigests map[basespec.Locator]cryptoutil.Digest `json:"-"`
}

// ManagedSkillDocument is the editable projection for a managed Skill.
// It deliberately contains the canonical SKILL.md document only. It never
// exposes Source configuration, a native filesystem path, or package internals.
type ManagedSkillDocument struct {
	Artifact artifact.Artifact
	Document agentskillsSpec.SkillDocument
}

type Bundle struct {
	Collection  collection.Collection
	Data        CollectionData
	Attachments []collection.Attachment
	Sources     []source.Summary
}

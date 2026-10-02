package topology

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/install/topology"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactory-go/model/source"
	"github.com/flexigpt/flexigpt-app/internal/jsonutil"
	"github.com/flexigpt/flexigpt-app/internal/yamlutil"
)

//go:embed builtin_topology.yaml
var builtinTopologyYAML []byte

const (
	BuiltinSourceRolePackages    = "packages"
	BuiltinEmbeddedPackageSkills = "skills"
	BuiltinEmbeddedPackageAgents = "agents"
	BuiltinEmbeddedPackageMCPs   = "mcps"
	BuiltinEmbeddedPackageTools  = "tools"
)

type ApplicationStorageKey string

const (
	ApplicationStorageDataDirectory              ApplicationStorageKey = "dataDirectory"
	ApplicationStorageSettingsDirectory          ApplicationStorageKey = "settingsDirectory"
	ApplicationStorageConversationsDirectory     ApplicationStorageKey = "conversationsDirectory"
	ApplicationStorageArtifactStoreDirectory     ApplicationStorageKey = "artifactStoreDirectory"
	ApplicationStorageArtifactStoreManifestFile  ApplicationStorageKey = "artifactStoreManifestFile"
	ApplicationStorageArtifactStoreMetadataFile  ApplicationStorageKey = "artifactStoreMetadataFile"
	ApplicationStorageArtifactStoreContentDir    ApplicationStorageKey = "artifactStoreContentDirectory"
	ApplicationStorageArtifactStoreStagingDir    ApplicationStorageKey = "artifactStoreStagingDirectory"
	ApplicationStorageArtifactStoreTemporaryName ApplicationStorageKey = "artifactStoreManifestTemporaryName"
)

type builtinRootWire struct {
	ID          root.RootID      `json:"id"`
	StorageKey  model.StorageKey `json:"storageKey"`
	DisplayName string           `json:"displayName"`
	Description string           `json:"description"`
	Protected   bool             `json:"protected"`
	Retained    bool             `json:"retained"`
}

type builtinSourceWire struct {
	Name        string               `json:"name"`
	Roles       []string             `json:"roles"`
	ID          source.SourceID      `json:"id"`
	StorageKey  model.StorageKey     `json:"storageKey"`
	Kind        source.SourceKind    `json:"kind"`
	DisplayName string               `json:"displayName"`
	Enabled     bool                 `json:"enabled"`
	Config      json.RawMessage      `json:"config"`
	Discovery   discoveryProfileWire `json:"discovery"`
}

type builtinTopologyWire struct {
	Application struct {
		Storage  map[string]string `json:"storage"`
		UserRoot builtinRootWire   `json:"userRoot"`
	} `json:"application"`

	Builtin struct {
		Root             builtinRootWire          `json:"root"`
		Sources          []builtinSourceWire      `json:"sources"`
		EmbeddedPackages map[string]model.Locator `json:"embeddedPackages"`
	} `json:"builtin"`
}

type builtinTopologyConfig struct {
	declaration          topology.Declaration
	sourcesByName        map[string]source.Draft
	sourceNameByRole     map[string]string
	embeddedPackageRoots map[string]model.Locator
	applicationStorage   map[ApplicationStorageKey]string

	userRoot         root.RootDraft
	userRootRetained bool
	builtinProtected bool
	builtinRetained  bool
}

var configuredBuiltinTopology = mustLoadBuiltinTopology(
	builtinTopologyYAML,
	configuredContractTopology,
)

func BuiltinTopologyDeclaration() topology.Declaration {
	return cloneBuiltinTopologyDeclaration(
		configuredBuiltinTopology.declaration,
	)
}

func BuiltinRootID() root.RootID {
	return configuredBuiltinTopology.declaration.Root.ID
}

func UserRootDraft() root.RootDraft {
	return configuredBuiltinTopology.userRoot
}

func UserRootID() root.RootID {
	return configuredBuiltinTopology.userRoot.ID
}

func ManagementRootIDs() []root.RootID {
	return []root.RootID{BuiltinRootID(), UserRootID()}
}

func BuiltinSource(
	role string,
) (source.Draft, error) {
	name, found := configuredBuiltinTopology.sourceNameByRole[role]
	if !found {
		return source.Draft{}, fmt.Errorf(
			"%w: built-in topology has no source role %q",
			model.ErrNotFound,
			role,
		)
	}
	value, found := configuredBuiltinTopology.sourcesByName[name]
	if !found {
		return source.Draft{}, fmt.Errorf(
			"%w: built-in topology source role %q is invalid",
			model.ErrInvalid,
			role,
		)
	}
	return cloneBuiltinSource(value), nil
}

func BuiltinPackageSourceID() source.SourceID {
	value, err := BuiltinSource(BuiltinSourceRolePackages)
	if err != nil {
		panic(err)
	}
	return value.ID
}

func IsBuiltinPackageSource(
	rootID root.RootID,
	sourceID source.SourceID,
) bool {
	return rootID == BuiltinRootID() &&
		sourceID == BuiltinPackageSourceID()
}

func BuiltinEmbeddedPackageRoot(
	name string,
) (model.Locator, error) {
	value, found := configuredBuiltinTopology.embeddedPackageRoots[name]
	if !found {
		return "", fmt.Errorf(
			"%w: built-in topology has no embedded package set %q",
			model.ErrNotFound,
			name,
		)
	}
	return value, nil
}

func MustBuiltinEmbeddedPackageRoot(name string) model.Locator {
	value, err := BuiltinEmbeddedPackageRoot(name)
	if err != nil {
		panic(err)
	}
	return value
}

func ApplicationStorageName(
	key ApplicationStorageKey,
) (string, error) {
	if key == "" {
		return "", fmt.Errorf(
			"%w: application storage key is required",
			model.ErrInvalid,
		)
	}
	value, found := configuredBuiltinTopology.applicationStorage[key]
	if !found {
		return "", fmt.Errorf(
			"%w: application topology has no storage entry %q",
			model.ErrNotFound,
			key,
		)
	}
	return value, nil
}

func MustApplicationStorageName(
	key ApplicationStorageKey,
) string {
	value, err := ApplicationStorageName(key)
	if err != nil {
		panic(err)
	}
	return value
}

func ApplicationStorageNames() []string {
	keys := make(
		[]ApplicationStorageKey,
		0,
		len(configuredBuiltinTopology.applicationStorage),
	)
	for key := range configuredBuiltinTopology.applicationStorage {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	output := make([]string, 0, len(keys))
	for _, key := range keys {
		output = append(
			output,
			configuredBuiltinTopology.applicationStorage[key],
		)
	}
	return output
}

func ProtectedRootIDs() []root.RootID {
	if !configuredBuiltinTopology.builtinProtected {
		return nil
	}
	return []root.RootID{BuiltinRootID()}
}

func RetainedRootDrafts() []root.RootDraft {
	output := make([]root.RootDraft, 0, 2)
	if configuredBuiltinTopology.builtinRetained {
		output = append(
			output,
			configuredBuiltinTopology.declaration.Root,
		)
	}
	if configuredBuiltinTopology.userRootRetained {
		output = append(
			output,
			configuredBuiltinTopology.userRoot,
		)
	}
	return output
}

func RetainedRootIDs() []root.RootID {
	drafts := RetainedRootDrafts()
	output := make([]root.RootID, 0, len(drafts))
	for _, draft := range drafts {
		output = append(output, draft.ID)
	}
	return output
}

func ValidateApplicationTopology() error {
	_, err := loadBuiltinTopology(
		builtinTopologyYAML,
		configuredContractTopology,
	)
	return err
}

func mustLoadBuiltinTopology(
	raw []byte,
	contract contractTopology,
) builtinTopologyConfig {
	value, err := loadBuiltinTopology(raw, contract)
	if err != nil {
		panic(fmt.Sprintf("invalid embedded builtin_topology.yaml: %v", err))
	}
	return value
}

func loadBuiltinTopology(
	raw []byte,
	contract contractTopology,
) (builtinTopologyConfig, error) {
	canonical, err := yamlutil.CanonicalObjectJSON(
		raw,
		model.MaxDefinitionBytes,
	)
	if err != nil {
		return builtinTopologyConfig{}, err
	}

	var wire builtinTopologyWire
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		&wire,
		model.MaxDefinitionBytes,
	); err != nil {
		return builtinTopologyConfig{}, err
	}

	userRoot := root.RootDraft{
		ID:          wire.Application.UserRoot.ID,
		StorageKey:  wire.Application.UserRoot.StorageKey,
		DisplayName: wire.Application.UserRoot.DisplayName,
		Description: wire.Application.UserRoot.Description,
	}
	if err := validateConfiguredRoot(
		"application user Root",
		userRoot,
	); err != nil {
		return builtinTopologyConfig{}, err
	}
	if wire.Application.UserRoot.Protected {
		return builtinTopologyConfig{}, fmt.Errorf(
			"%w: application user Root cannot be protected",
			model.ErrInvalid,
		)
	}
	if !wire.Application.UserRoot.Retained {
		return builtinTopologyConfig{}, fmt.Errorf(
			"%w: application user Root must be retained",
			model.ErrInvalid,
		)
	}

	storage, err := parseApplicationStorage(
		wire.Application.Storage,
	)
	if err != nil {
		return builtinTopologyConfig{}, err
	}
	if len(wire.Builtin.Sources) == 0 {
		return builtinTopologyConfig{}, fmt.Errorf(
			"%w: built-in topology has no Sources",
			model.ErrInvalid,
		)
	}

	declaration := topology.Declaration{
		Root: root.RootDraft{
			ID:          wire.Builtin.Root.ID,
			StorageKey:  wire.Builtin.Root.StorageKey,
			DisplayName: wire.Builtin.Root.DisplayName,
			Description: wire.Builtin.Root.Description,
		},
		Sources: make([]source.Draft, 0, len(wire.Builtin.Sources)),
	}
	sourcesByName := make(map[string]source.Draft, len(wire.Builtin.Sources))
	sourceNameByRole := make(map[string]string)

	for index, value := range wire.Builtin.Sources {
		if err := model.ValidateIdentifier(
			"built-in source name",
			value.Name,
			model.MaxKindBytes,
		); err != nil {
			return builtinTopologyConfig{}, fmt.Errorf(
				"builtin sources[%d]: %w",
				index,
				err,
			)
		}
		if _, duplicate := sourcesByName[value.Name]; duplicate {
			return builtinTopologyConfig{}, fmt.Errorf(
				"%w: built-in topology repeats source name %q",
				model.ErrInvalid,
				value.Name,
			)
		}

		config := append(json.RawMessage(nil), value.Config...)
		if len(config) == 0 {
			config = json.RawMessage(jsonutil.EmptyObject)
		}
		config, err = jsonutil.CanonicalizeObject(
			config,
			model.MaxDefinitionBytes,
		)
		if err != nil {
			return builtinTopologyConfig{}, fmt.Errorf(
				"builtin source %q config: %w",
				value.Name,
				err,
			)
		}
		discovery, err := parseDiscoveryProfile(
			"builtin source "+value.Name,
			value.Discovery,
			contract.documentSets,
		)
		if err != nil {
			return builtinTopologyConfig{}, err
		}

		draft := source.Draft{
			ID:          value.ID,
			StorageKey:  value.StorageKey,
			Kind:        value.Kind,
			DisplayName: value.DisplayName,
			Enabled:     value.Enabled,
			Config:      config,
			Discovery:   discovery,
		}
		declaration.Sources = append(declaration.Sources, draft)
		sourcesByName[value.Name] = cloneBuiltinSource(draft)

		seenRoles := make(map[string]struct{}, len(value.Roles))
		for _, role := range value.Roles {
			if err := model.ValidateIdentifier(
				"built-in source role",
				role,
				model.MaxKindBytes,
			); err != nil {
				return builtinTopologyConfig{}, err
			}
			if _, duplicate := seenRoles[role]; duplicate {
				return builtinTopologyConfig{}, fmt.Errorf(
					"%w: source %q repeats role %q",
					model.ErrInvalid,
					value.Name,
					role,
				)
			}
			seenRoles[role] = struct{}{}
			if previous, duplicate := sourceNameByRole[role]; duplicate {
				return builtinTopologyConfig{}, fmt.Errorf(
					"%w: source role %q is declared by both %q and %q",
					model.ErrInvalid,
					role,
					previous,
					value.Name,
				)
			}
			sourceNameByRole[role] = value.Name
		}
	}

	if err := declaration.Validate(); err != nil {
		return builtinTopologyConfig{}, err
	}
	if userRoot.ID == declaration.Root.ID ||
		userRoot.StorageKey == declaration.Root.StorageKey {
		return builtinTopologyConfig{}, fmt.Errorf(
			"%w: built-in and user Roots must have distinct identities and storage keys",
			model.ErrConflict,
		)
	}
	for _, sourceValue := range declaration.Sources {
		if declaration.Root.ID == root.RootID(sourceValue.ID) ||
			userRoot.ID == root.RootID(sourceValue.ID) {
			return builtinTopologyConfig{}, fmt.Errorf(
				"%w: application Root and Source IDs must differ",
				model.ErrConflict,
			)
		}
	}
	packageSourceName, found := sourceNameByRole[BuiltinSourceRolePackages]
	if !found {
		return builtinTopologyConfig{}, fmt.Errorf(
			"%w: built-in topology has no package Source role",
			model.ErrInvalid,
		)
	}
	packageSource := sourcesByName[packageSourceName]
	if packageSource.Kind != source.SourceKindManagedDirectory {
		return builtinTopologyConfig{}, fmt.Errorf(
			"%w: built-in package Source must be managed",
			model.ErrInvalid,
		)
	}

	embeddedRoots := make(
		map[string]model.Locator,
		len(wire.Builtin.EmbeddedPackages),
	)
	for name, locator := range wire.Builtin.EmbeddedPackages {
		if err := model.ValidateIdentifier(
			"embedded package set name",
			name,
			model.MaxKindBytes,
		); err != nil {
			return builtinTopologyConfig{}, err
		}
		if err := locator.ValidatePortable(false); err != nil {
			return builtinTopologyConfig{}, err
		}
		embeddedRoots[name] = locator
	}
	for _, required := range []string{
		BuiltinEmbeddedPackageSkills,
		BuiltinEmbeddedPackageAgents,
		BuiltinEmbeddedPackageMCPs,
		BuiltinEmbeddedPackageTools,
	} {
		if _, found := embeddedRoots[required]; !found {
			return builtinTopologyConfig{}, fmt.Errorf(
				"%w: built-in topology has no embedded package set %q",
				model.ErrInvalid,
				required,
			)
		}
	}

	return builtinTopologyConfig{
		declaration:          cloneBuiltinTopologyDeclaration(declaration),
		sourcesByName:        sourcesByName,
		sourceNameByRole:     sourceNameByRole,
		embeddedPackageRoots: embeddedRoots,
		applicationStorage:   storage,
		userRoot:             userRoot,
		userRootRetained:     wire.Application.UserRoot.Retained,
		builtinProtected:     wire.Builtin.Root.Protected,
		builtinRetained:      wire.Builtin.Root.Retained,
	}, nil
}

func validateConfiguredRoot(
	label string,
	value root.RootDraft,
) error {
	if err := value.ID.Validate(); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if err := value.StorageKey.Validate(); err != nil {
		return fmt.Errorf("%s: %w", label, err)
	}
	if err := model.ValidateRequiredText(
		label+" display name",
		value.DisplayName,
		model.MaxDisplayNameBytes,
	); err != nil {
		return err
	}
	return model.ValidateOptionalText(
		label+" description",
		value.Description,
		model.MaxDescriptionBytes,
	)
}

func parseApplicationStorage(
	values map[string]string,
) (map[ApplicationStorageKey]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: application storage declarations are required",
			model.ErrInvalid,
		)
	}

	seenValues := make(map[string]ApplicationStorageKey, len(values))
	output := make(map[ApplicationStorageKey]string, len(values))
	for rawKey, name := range values {
		key := ApplicationStorageKey(rawKey)
		if err := model.ValidateIdentifier(
			"application storage key",
			rawKey,
			model.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		if err := model.ValidateRequiredText(
			"application storage name",
			name,
			model.MaxLogicalNameBytes,
		); err != nil {
			return nil, err
		}
		if strings.HasPrefix(name, ".") || strings.Contains(name, "/") {
			return nil, fmt.Errorf(
				"%w: application storage name %q is invalid",
				model.ErrInvalid,
				name,
			)
		}
		if previous, duplicate := seenValues[name]; duplicate {
			return nil, fmt.Errorf(
				"%w: application storage keys %q and %q both use %q",
				model.ErrInvalid,
				previous,
				key,
				name,
			)
		}
		seenValues[name] = key
		output[key] = name
	}

	for _, key := range []ApplicationStorageKey{
		ApplicationStorageDataDirectory,
		ApplicationStorageSettingsDirectory,
		ApplicationStorageConversationsDirectory,
		ApplicationStorageArtifactStoreDirectory,
		ApplicationStorageArtifactStoreManifestFile,
		ApplicationStorageArtifactStoreMetadataFile,
		ApplicationStorageArtifactStoreContentDir,
		ApplicationStorageArtifactStoreStagingDir,
		ApplicationStorageArtifactStoreTemporaryName,
	} {
		if _, found := output[key]; found {
			continue
		}
		return nil, fmt.Errorf(
			"%w: application storage declaration %q is required",
			model.ErrInvalid,
			key,
		)
	}
	return output, nil
}

func cloneBuiltinTopologyDeclaration(
	value topology.Declaration,
) topology.Declaration {
	output := value
	output.Sources = make([]source.Draft, len(value.Sources))
	for index, sourceValue := range value.Sources {
		output.Sources[index] = cloneBuiltinSource(sourceValue)
	}
	return output
}

func cloneBuiltinSource(value source.Draft) source.Draft {
	output := value
	output.Config = append(json.RawMessage(nil), value.Config...)
	output.Discovery = value.Discovery.Clone()
	return output
}

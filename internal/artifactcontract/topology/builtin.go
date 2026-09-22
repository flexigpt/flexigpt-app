package topology

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/root"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/basespec/source"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/installerapi/topology"
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
)

type ApplicationStorageKey string

const (
	ApplicationStorageDataDirectory              ApplicationStorageKey = "dataDirectory"
	ApplicationStorageSettingsDirectory          ApplicationStorageKey = "settingsDirectory"
	ApplicationStorageConversationsDirectory     ApplicationStorageKey = "conversationsDirectory"
	ApplicationStorageModelPresetsDirectory      ApplicationStorageKey = "modelPresetsDirectory"
	ApplicationStorageToolsDirectory             ApplicationStorageKey = "toolsDirectory"
	ApplicationStorageArtifactStoreDirectory     ApplicationStorageKey = "artifactStoreDirectory"
	ApplicationStorageArtifactStoreManifestFile  ApplicationStorageKey = "artifactStoreManifestFile"
	ApplicationStorageArtifactStoreMetadataFile  ApplicationStorageKey = "artifactStoreMetadataFile"
	ApplicationStorageArtifactStoreContentDir    ApplicationStorageKey = "artifactStoreContentDirectory"
	ApplicationStorageArtifactStoreStagingDir    ApplicationStorageKey = "artifactStoreStagingDirectory"
	ApplicationStorageArtifactStoreTemporaryName ApplicationStorageKey = "artifactStoreManifestTemporaryName"
)

type builtinRootWire struct {
	ID          root.RootID         `json:"id"`
	StorageKey  basespec.StorageKey `json:"storageKey"`
	DisplayName string              `json:"displayName"`
	Description string              `json:"description"`
	Protected   bool                `json:"protected"`
	Retained    bool                `json:"retained"`
}

type builtinSourceWire struct {
	Name        string               `json:"name"`
	Roles       []string             `json:"roles"`
	ID          source.SourceID      `json:"id"`
	StorageKey  basespec.StorageKey  `json:"storageKey"`
	Kind        source.SourceKind    `json:"kind"`
	DisplayName string               `json:"displayName"`
	Enabled     bool                 `json:"enabled"`
	Config      json.RawMessage      `json:"config"`
	Discovery   discoveryProfileWire `json:"discovery"`
}

type builtinTopologyWire struct {
	Application struct {
		Storage map[string]string `json:"storage"`
	} `json:"application"`

	Builtin struct {
		Root             builtinRootWire             `json:"root"`
		Sources          []builtinSourceWire         `json:"sources"`
		EmbeddedPackages map[string]basespec.Locator `json:"embeddedPackages"`
	} `json:"builtin"`
}

type builtinTopologyConfig struct {
	declaration          topology.Declaration
	sourcesByName        map[string]source.Draft
	sourceNameByRole     map[string]string
	embeddedPackageRoots map[string]basespec.Locator
	applicationStorage   map[ApplicationStorageKey]string
	protected            bool
	retained             bool
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

func BuiltinSource(
	role string,
) (source.Draft, error) {
	name, found := configuredBuiltinTopology.sourceNameByRole[role]
	if !found {
		return source.Draft{}, fmt.Errorf(
			"%w: built-in topology has no source role %q",
			basespec.ErrNotFound,
			role,
		)
	}
	value, found := configuredBuiltinTopology.sourcesByName[name]
	if !found {
		return source.Draft{}, fmt.Errorf(
			"%w: built-in topology source role %q is invalid",
			basespec.ErrInvalid,
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
) (basespec.Locator, error) {
	value, found := configuredBuiltinTopology.embeddedPackageRoots[name]
	if !found {
		return "", fmt.Errorf(
			"%w: built-in topology has no embedded package set %q",
			basespec.ErrNotFound,
			name,
		)
	}
	return value, nil
}

func MustBuiltinEmbeddedPackageRoot(name string) basespec.Locator {
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
			basespec.ErrInvalid,
		)
	}
	value, found := configuredBuiltinTopology.applicationStorage[key]
	if !found {
		return "", fmt.Errorf(
			"%w: application topology has no storage entry %q",
			basespec.ErrNotFound,
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
	if !configuredBuiltinTopology.protected {
		return nil
	}
	return []root.RootID{BuiltinRootID()}
}

func RetainedRootDrafts() []root.RootDraft {
	if !configuredBuiltinTopology.retained {
		return nil
	}
	return []root.RootDraft{
		configuredBuiltinTopology.declaration.Root,
	}
}

func RetainedRootIDs() []root.RootID {
	if !configuredBuiltinTopology.retained {
		return nil
	}
	return []root.RootID{BuiltinRootID()}
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
		basespec.MaxDefinitionBytes,
	)
	if err != nil {
		return builtinTopologyConfig{}, err
	}

	var wire builtinTopologyWire
	if err := jsonutil.DecodeCanonicalObjectBytesInto(
		canonical,
		&wire,
		basespec.MaxDefinitionBytes,
	); err != nil {
		return builtinTopologyConfig{}, err
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
			basespec.ErrInvalid,
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
		if err := basespec.ValidateIdentifier(
			"built-in source name",
			value.Name,
			basespec.MaxKindBytes,
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
				basespec.ErrInvalid,
				value.Name,
			)
		}

		config := append(json.RawMessage(nil), value.Config...)
		if len(config) == 0 {
			config = json.RawMessage(jsonutil.EmptyObject)
		}
		config, err = jsonutil.CanonicalizeObject(
			config,
			basespec.MaxDefinitionBytes,
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
			if err := basespec.ValidateIdentifier(
				"built-in source role",
				role,
				basespec.MaxKindBytes,
			); err != nil {
				return builtinTopologyConfig{}, err
			}
			if _, duplicate := seenRoles[role]; duplicate {
				return builtinTopologyConfig{}, fmt.Errorf(
					"%w: source %q repeats role %q",
					basespec.ErrInvalid,
					value.Name,
					role,
				)
			}
			seenRoles[role] = struct{}{}
			if previous, duplicate := sourceNameByRole[role]; duplicate {
				return builtinTopologyConfig{}, fmt.Errorf(
					"%w: source role %q is declared by both %q and %q",
					basespec.ErrInvalid,
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
	for _, sourceValue := range declaration.Sources {
		if declaration.Root.ID == root.RootID(sourceValue.ID) {
			return builtinTopologyConfig{}, fmt.Errorf(
				"%w: built-in Root and Source IDs must differ",
				basespec.ErrConflict,
			)
		}
	}
	packageSourceName, found := sourceNameByRole[BuiltinSourceRolePackages]
	if !found {
		return builtinTopologyConfig{}, fmt.Errorf(
			"%w: built-in topology has no package Source role",
			basespec.ErrInvalid,
		)
	}
	packageSource := sourcesByName[packageSourceName]
	if packageSource.Kind != source.SourceKindManagedDirectory {
		return builtinTopologyConfig{}, fmt.Errorf(
			"%w: built-in package Source must be managed",
			basespec.ErrInvalid,
		)
	}

	embeddedRoots := make(
		map[string]basespec.Locator,
		len(wire.Builtin.EmbeddedPackages),
	)
	for name, locator := range wire.Builtin.EmbeddedPackages {
		if err := basespec.ValidateIdentifier(
			"embedded package set name",
			name,
			basespec.MaxKindBytes,
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
	} {
		if _, found := embeddedRoots[required]; !found {
			return builtinTopologyConfig{}, fmt.Errorf(
				"%w: built-in topology has no embedded package set %q",
				basespec.ErrInvalid,
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
		protected:            wire.Builtin.Root.Protected,
		retained:             wire.Builtin.Root.Retained,
	}, nil
}

func parseApplicationStorage(
	values map[string]string,
) (map[ApplicationStorageKey]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf(
			"%w: application storage declarations are required",
			basespec.ErrInvalid,
		)
	}

	seenValues := make(map[string]ApplicationStorageKey, len(values))
	output := make(map[ApplicationStorageKey]string, len(values))
	for rawKey, name := range values {
		key := ApplicationStorageKey(rawKey)
		if err := basespec.ValidateIdentifier(
			"application storage key",
			rawKey,
			basespec.MaxKindBytes,
		); err != nil {
			return nil, err
		}
		if err := basespec.ValidateRequiredText(
			"application storage name",
			name,
			basespec.MaxLogicalNameBytes,
		); err != nil {
			return nil, err
		}
		if strings.HasPrefix(name, ".") || strings.Contains(name, "/") {
			return nil, fmt.Errorf(
				"%w: application storage name %q is invalid",
				basespec.ErrInvalid,
				name,
			)
		}
		if previous, duplicate := seenValues[name]; duplicate {
			return nil, fmt.Errorf(
				"%w: application storage keys %q and %q both use %q",
				basespec.ErrInvalid,
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
		ApplicationStorageModelPresetsDirectory,
		ApplicationStorageToolsDirectory,
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
			basespec.ErrInvalid,
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

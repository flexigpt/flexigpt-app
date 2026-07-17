package contentstore

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
	"github.com/flexigpt/mapstore-go"
	"github.com/flexigpt/mapstore-go/jsonencdec"
)

const (
	portableAssetFileFormatV1 = "artifact-asset/v1"
)

type mapStorePortableContentRepository struct {
	store  *mapstore.MapDirectoryStore
	mu     sync.Mutex
	closed bool
}

type artifactContentPartition string

const (
	artifactContentDefinitions artifactContentPartition = "definitions"
	artifactContentAssets      artifactContentPartition = "assets"
	artifactContentPackages    artifactContentPartition = "packages"
)

type portableAssetFile struct {
	Format    string `json:"format"`
	Digest    string `json:"digest"`
	SizeBytes int64  `json:"sizeBytes"`
	Data      string `json:"data"`
}

type artifactContentPartitionProvider struct{}

func (artifactContentPartitionProvider) GetPartitionDir(key mapstore.FileKey) (string, error) {
	partition, ok := key.XAttr.(artifactContentPartition)
	if !ok || partition == "" {
		return "", fmt.Errorf("%w: missing portable content partition", spec.ErrInvalidRequest)
	}
	switch partition {
	case artifactContentDefinitions, artifactContentAssets, artifactContentPackages:
		return string(partition), nil
	default:
		return "", fmt.Errorf("%w: invalid portable content partition %q", spec.ErrInvalidRequest, partition)
	}
}

func (artifactContentPartitionProvider) ListPartitions(
	_, sortOrder, _ string,
	_ int,
) (dirs []string, nextPageToken string, err error) {
	partitions := []string{
		string(artifactContentAssets),
		string(artifactContentDefinitions),
		string(artifactContentPackages),
	}
	if strings.EqualFold(sortOrder, mapstore.SortOrderDescending) {
		sort.Sort(sort.Reverse(sort.StringSlice(partitions)))
	} else {
		sort.Strings(partitions)
	}
	return partitions, "", nil
}

// NewMapStorePortableContentRepository creates the approved MapStore-backed
// storage facade for shareable JSON definitions, assets, and package manifests.
func NewMapStorePortableContentRepository(baseDir string) (spec.PortableContentRepository, error) {
	store, err := mapstore.NewMapDirectoryStore(
		baseDir,
		true,
		artifactContentPartitionProvider{},
		jsonencdec.JSONEncoderDecoder{},
	)
	if err != nil {
		return nil, fmt.Errorf("open MapStore portable content repository: %w", err)
	}
	return &mapStorePortableContentRepository{store: store}, nil
}

func (r *mapStorePortableContentRepository) Close() error {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.store == nil {
		return nil
	}
	r.closed = true
	store := r.store
	r.store = nil
	return store.CloseAll()
}

func (r *mapStorePortableContentRepository) PutDefinition(
	ctx context.Context,
	file spec.ArtifactDefinitionFile,
) (spec.CanonicalDefinition, error) {
	if err := ctx.Err(); err != nil {
		return spec.CanonicalDefinition{}, err
	}
	if file.Format == "" {
		file.Format = spec.ArtifactDefinitionFileFormatV1
	}
	normalized, err := baseutils.CanonicalizeDefinition(file.Definition)
	if err != nil {
		return spec.CanonicalDefinition{}, err
	}
	file.Definition = normalized
	if err := validate.ValidateArtifactDefinitionFile(file); err != nil {
		return spec.CanonicalDefinition{}, err
	}
	key, err := definitionFileKey(normalized.Digest)
	if err != nil {
		return spec.CanonicalDefinition{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureOpenLocked(); err != nil {
		return spec.CanonicalDefinition{}, err
	}
	if err := ctx.Err(); err != nil {
		return spec.CanonicalDefinition{}, err
	}
	data, err := jsonencdec.StructWithJSONTagsToMap(file)
	if err != nil {
		return spec.CanonicalDefinition{}, fmt.Errorf("encode portable definition: %w", err)
	}
	if err := r.store.SetFileData(key, data); err != nil {
		return spec.CanonicalDefinition{}, fmt.Errorf("persist portable definition: %w", err)
	}
	stored, err := r.getDefinitionLocked(ctx, normalized.Digest)
	if err != nil {
		return spec.CanonicalDefinition{}, err
	}
	if stored.Digest != normalized.Digest {
		return spec.CanonicalDefinition{}, fmt.Errorf(
			"%w: persisted definition %q",
			spec.ErrDigestMismatch,
			normalized.Digest,
		)
	}
	return stored, nil
}

func (r *mapStorePortableContentRepository) GetDefinition(
	ctx context.Context,
	digest spec.Digest,
) (spec.CanonicalDefinition, error) {
	if err := ctx.Err(); err != nil {
		return spec.CanonicalDefinition{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureOpenLocked(); err != nil {
		return spec.CanonicalDefinition{}, err
	}
	return r.getDefinitionLocked(ctx, digest)
}

func (r *mapStorePortableContentRepository) PutAsset(ctx context.Context, content []byte) (spec.Digest, int64, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	if int64(len(content)) > spec.MaxTransferPayloadBytes {
		return "", 0, fmt.Errorf(
			"%w: asset exceeds %d bytes",
			spec.ErrInvalidRequest,
			spec.MaxTransferPayloadBytes,
		)
	}
	digest := baseutils.DigestBytes(content)
	key, err := assetFileKey(digest)
	if err != nil {
		return "", 0, err
	}
	file := portableAssetFile{
		Format:    portableAssetFileFormatV1,
		Digest:    string(digest),
		SizeBytes: int64(len(content)),
		Data:      base64.StdEncoding.EncodeToString(content),
	}
	blob, err := jsonencdec.StructWithJSONTagsToMap(file)
	if err != nil {
		return "", 0, fmt.Errorf("encode portable asset: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureOpenLocked(); err != nil {
		return "", 0, err
	}
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	if err := r.store.SetFileData(key, blob); err != nil {
		return "", 0, fmt.Errorf("persist portable asset: %w", err)
	}
	return digest, int64(len(content)), nil
}

func (r *mapStorePortableContentRepository) GetAsset(ctx context.Context, digest spec.Digest) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key, err := assetFileKey(digest)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureOpenLocked(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := r.store.GetFileData(key, true)
	if err != nil {
		return nil, fmt.Errorf("%w: asset %q: %w", spec.ErrContentNotFound, digest, err)
	}
	var file portableAssetFile
	if err := jsonencdec.MapToStructWithJSONTags(data, &file); err != nil {
		return nil, fmt.Errorf("decode portable asset %q: %w", digest, err)
	}
	if file.Format != portableAssetFileFormatV1 ||
		spec.Digest(file.Digest) != digest ||
		file.SizeBytes < 0 ||
		file.SizeBytes > spec.MaxTransferPayloadBytes {
		return nil, fmt.Errorf("%w: malformed portable asset %q", spec.ErrInvalidRequest, digest)
	}
	decoded, err := base64.StdEncoding.DecodeString(file.Data)
	if err != nil {
		return nil, fmt.Errorf("decode portable asset %q: %w", digest, err)
	}
	if int64(len(decoded)) != file.SizeBytes {
		return nil, fmt.Errorf(
			"%w: asset %q stored size is %d, decoded size is %d",
			spec.ErrDigestMismatch,
			digest,
			file.SizeBytes,
			len(decoded),
		)
	}
	if baseutils.DigestBytes(decoded) != digest {
		return nil, fmt.Errorf("%w: asset %q", spec.ErrDigestMismatch, digest)
	}
	return decoded, nil
}

func (r *mapStorePortableContentRepository) PutPackageManifest(
	ctx context.Context,
	keyValue string,
	manifest spec.PortablePackageManifest,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validate.ValidatePortablePackageManifest(manifest); err != nil {
		return err
	}
	key, err := packageManifestFileKey(keyValue)
	if err != nil {
		return err
	}
	data, err := jsonencdec.StructWithJSONTagsToMap(manifest)
	if err != nil {
		return fmt.Errorf("encode portable package manifest: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureOpenLocked(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.store.SetFileData(key, data); err != nil {
		return fmt.Errorf("persist portable package manifest: %w", err)
	}
	return nil
}

func (r *mapStorePortableContentRepository) GetPackageManifest(
	ctx context.Context,
	keyValue string,
) (spec.PortablePackageManifest, error) {
	if err := ctx.Err(); err != nil {
		return spec.PortablePackageManifest{}, err
	}
	key, err := packageManifestFileKey(keyValue)
	if err != nil {
		return spec.PortablePackageManifest{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.ensureOpenLocked(); err != nil {
		return spec.PortablePackageManifest{}, err
	}
	if err := ctx.Err(); err != nil {
		return spec.PortablePackageManifest{}, err
	}
	data, err := r.store.GetFileData(key, true)
	if err != nil {
		return spec.PortablePackageManifest{}, fmt.Errorf(
			"%w: package manifest %q: %w",
			spec.ErrContentNotFound,
			keyValue,
			err,
		)
	}
	var manifest spec.PortablePackageManifest
	if err := jsonencdec.MapToStructWithJSONTags(data, &manifest); err != nil {
		return spec.PortablePackageManifest{}, fmt.Errorf("decode portable package manifest: %w", err)
	}
	if err := validate.ValidatePortablePackageManifest(manifest); err != nil {
		return spec.PortablePackageManifest{}, err
	}
	return manifest, nil
}

func (r *mapStorePortableContentRepository) getDefinitionLocked(
	ctx context.Context,
	digest spec.Digest,
) (def spec.CanonicalDefinition, err error) {
	if err := r.ensureOpenLocked(); err != nil {
		return spec.CanonicalDefinition{}, err
	}
	if err := ctx.Err(); err != nil {
		return spec.CanonicalDefinition{}, err
	}
	key, err := definitionFileKey(digest)
	if err != nil {
		return spec.CanonicalDefinition{}, err
	}
	data, err := r.store.GetFileData(key, true)
	if err != nil {
		return spec.CanonicalDefinition{}, fmt.Errorf("%w: definition %q: %w", spec.ErrContentNotFound, digest, err)
	}
	var file spec.ArtifactDefinitionFile
	if err := jsonencdec.MapToStructWithJSONTags(data, &file); err != nil {
		return spec.CanonicalDefinition{}, fmt.Errorf("decode portable definition %q: %w", digest, err)
	}
	if err := validate.ValidateArtifactDefinitionFile(file); err != nil {
		return spec.CanonicalDefinition{}, fmt.Errorf("validate portable definition %q: %w", digest, err)
	}
	normalized, err := baseutils.CanonicalizeDefinition(file.Definition)
	if err != nil {
		return spec.CanonicalDefinition{}, err
	}
	if normalized.Digest != digest {
		return spec.CanonicalDefinition{}, fmt.Errorf(
			"%w: requested %q, stored %q",
			spec.ErrDigestMismatch,
			digest,
			normalized.Digest,
		)
	}
	return normalized, nil
}

func (r *mapStorePortableContentRepository) ensureOpenLocked() error {
	if r == nil || r.closed || r.store == nil {
		return spec.ErrClosed
	}
	return nil
}

func definitionFileKey(digest spec.Digest) (mapstore.FileKey, error) {
	hexDigest, err := portableDigestHex(digest)
	if err != nil {
		return mapstore.FileKey{}, err
	}
	return mapstore.FileKey{FileName: "definition-" + hexDigest + ".json", XAttr: artifactContentDefinitions}, nil
}

func assetFileKey(digest spec.Digest) (mapstore.FileKey, error) {
	hexDigest, err := portableDigestHex(digest)
	if err != nil {
		return mapstore.FileKey{}, err
	}
	return mapstore.FileKey{FileName: "asset-" + hexDigest + ".json", XAttr: artifactContentAssets}, nil
}

func packageManifestFileKey(keyValue string) (mapstore.FileKey, error) {
	if strings.TrimSpace(keyValue) == "" {
		return mapstore.FileKey{}, fmt.Errorf("%w: package manifest key is empty", spec.ErrInvalidRequest)
	}
	digest := baseutils.DigestBytes([]byte(keyValue))
	hexDigest, err := portableDigestHex(digest)
	if err != nil {
		return mapstore.FileKey{}, err
	}
	return mapstore.FileKey{FileName: "package-" + hexDigest + ".json", XAttr: artifactContentPackages}, nil
}

func portableDigestHex(digest spec.Digest) (string, error) {
	value := string(digest)
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+64 {
		return "", fmt.Errorf("%w: invalid digest %q", spec.ErrInvalidRequest, digest)
	}
	hexValue := strings.TrimPrefix(value, "sha256:")
	if _, err := hex.DecodeString(hexValue); err != nil || strings.ToLower(hexValue) != hexValue {
		return "", fmt.Errorf("%w: invalid digest %q", spec.ErrInvalidRequest, digest)
	}
	return hexValue, nil
}

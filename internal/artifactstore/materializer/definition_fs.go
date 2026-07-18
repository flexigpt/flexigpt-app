package materializer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/flexigpt/llmtools-go/fstool"
	llmtoolsSpec "github.com/flexigpt/llmtools-go/spec"

	"github.com/flexigpt/flexigpt-app/internal/artifactstore/baseutils"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/spec"
	"github.com/flexigpt/flexigpt-app/internal/artifactstore/validate"
)

const (
	fsDefinitionReceiptFormat = "artifactstore.fs-definition-receipt/v1"
	binaryEncoding            = "binary"
)

type FSDefinitionMaterializer struct{}

type fsMaterializedFile struct {
	Locator spec.SourceLocator `json:"locator"`
	Digest  spec.Digest        `json:"digest"`
}

type fsDefinitionReceipt struct {
	Format       string               `json:"format"`
	SourceID     spec.SourceID        `json:"sourceID"`
	ConfigDigest spec.Digest          `json:"configDigest"`
	Files        []fsMaterializedFile `json:"files"`
}

type fsDefinitionFile struct {
	Locator spec.SourceLocator
	Content []byte
	Digest  spec.Digest
}

func NewFSDefinitionMaterializer() *FSDefinitionMaterializer {
	return &FSDefinitionMaterializer{}
}

func (*FSDefinitionMaterializer) Kind() spec.SourceKind {
	return spec.SourceKindFSDirectory
}

func (*FSDefinitionMaterializer) MaterializeDefinition(
	ctx context.Context,
	request spec.DefinitionMaterializationRequest,
) (spec.DefinitionMaterialization, error) {
	if err := ctx.Err(); err != nil {
		return spec.DefinitionMaterialization{}, err
	}
	if request.Source.Kind != spec.SourceKindFSDirectory {
		return spec.DefinitionMaterialization{}, fmt.Errorf(
			"%w: filesystem materializer received source kind %q",
			spec.ErrInvalidRequest,
			request.Source.Kind,
		)
	}
	if !request.Exclusive {
		return spec.DefinitionMaterialization{}, fmt.Errorf(
			"%w: filesystem definition materialization requires exclusive publication",
			spec.ErrInvalidRequest,
		)
	}
	if request.Destination.FrontendID != "" &&
		request.Destination.FrontendID != spec.PortableDefinitionFrontendID {
		return spec.DefinitionMaterialization{}, fmt.Errorf(
			"%w: filesystem materializer writes portable definition envelopes only",
			spec.ErrUnsupported,
		)
	}

	tool, config, configDigest, err := fsToolForManagedSource(request.Source)
	if err != nil {
		return spec.DefinitionMaterialization{}, err
	}
	files, rootSourceDigest, err := buildFSDefinitionFiles(
		request.Source.SourceID,
		request.Destination,
		request.Payload,
	)
	if err != nil {
		return spec.DefinitionMaterialization{}, err
	}

	written := make([]fsMaterializedFile, 0, len(files))
	for _, file := range files {
		_, err := tool.WriteFile(ctx, fstool.WriteFileArgs{
			Path:          string(file.Locator),
			Encoding:      binaryEncoding,
			Content:       base64.StdEncoding.EncodeToString(file.Content),
			Overwrite:     false,
			CreateParents: true,
		})
		if err != nil {
			rollbackErr := discardFSMaterializedFiles(
				context.WithoutCancel(ctx),
				tool,
				config,
				written,
			)
			return spec.DefinitionMaterialization{}, errors.Join(err, rollbackErr)
		}
		written = append(written, fsMaterializedFile{
			Locator: file.Locator,
			Digest:  file.Digest,
		})
	}

	receipt, err := encodeFSDefinitionReceipt(fsDefinitionReceipt{
		Format:       fsDefinitionReceiptFormat,
		SourceID:     request.Source.SourceID,
		ConfigDigest: configDigest,
		Files:        written,
	})
	if err != nil {
		rollbackErr := discardFSMaterializedFiles(
			context.WithoutCancel(ctx),
			tool,
			config,
			written,
		)
		return spec.DefinitionMaterialization{}, errors.Join(err, rollbackErr)
	}
	return spec.DefinitionMaterialization{
		SourceContentDigest: rootSourceDigest,
		Receipt:             receipt,
	}, nil
}

func (*FSDefinitionMaterializer) DiscardDefinition(
	ctx context.Context,
	source spec.ArtifactSource,
	receiptValue string,
) error {
	receipt, err := decodeFSDefinitionReceipt(receiptValue)
	if err != nil {
		return err
	}
	if receipt.SourceID != source.SourceID {
		return fmt.Errorf(
			"%w: definition receipt belongs to source %q, not %q",
			spec.ErrInvalidRequest,
			receipt.SourceID,
			source.SourceID,
		)
	}
	tool, config, configDigest, err := fsToolForManagedSource(source)
	if err != nil {
		return err
	}
	if receipt.ConfigDigest != configDigest {
		return fmt.Errorf(
			"%w: source configuration changed after definition materialization",
			spec.ErrConflict,
		)
	}
	return discardFSMaterializedFiles(ctx, tool, config, receipt.Files)
}

func buildFSDefinitionFiles(
	sourceID spec.SourceID,
	destination spec.TransferDestination,
	payload spec.DefinitionTransferPayload,
) ([]fsDefinitionFile, spec.Digest, error) {
	destinationDir := path.Dir(string(destination.Locator))
	files := make(map[spec.SourceLocator]fsDefinitionFile)
	var rootSourceDigest spec.Digest
	rootSeen := false

	add := func(locator spec.SourceLocator, content []byte) error {
		if err := validate.ValidateCatalogResourceKey(spec.CatalogResourceKey{
			SourceID: sourceID,
			Locator:  locator,
		}); err != nil {
			return err
		}
		if _, exists := files[locator]; exists {
			return fmt.Errorf(
				"%w: multiple transfer files resolve to destination %q",
				spec.ErrInvalidRequest,
				locator,
			)
		}
		files[locator] = fsDefinitionFile{
			Locator: locator,
			Content: append([]byte(nil), content...),
			Digest:  baseutils.DigestBytes(content),
		}
		return nil
	}

	for _, definitionFile := range payload.Definitions {
		if definitionFile.Format == "" {
			definitionFile.Format = spec.ArtifactDefinitionFileFormatV1
		}
		canonical, err := baseutils.CanonicalizeDefinition(definitionFile.Definition)
		if err != nil {
			return nil, "", err
		}
		definitionFile.Definition = canonical
		if err := validate.ValidateArtifactDefinitionFile(definitionFile); err != nil {
			return nil, "", err
		}
		raw, err := json.Marshal(definitionFile)
		if err != nil {
			return nil, "", err
		}
		raw, err = baseutils.CanonicalizeJSON(raw)
		if err != nil {
			return nil, "", err
		}

		locator := destination.Locator
		if canonical.Digest != payload.RootDefinitionDigest {
			hexDigest := strings.TrimPrefix(string(canonical.Digest), "sha256:")
			locator = spec.SourceLocator(path.Join(
				destinationDir,
				spec.ManagedArtifactDefinitionsDirectoryName,
				"definition-"+hexDigest+".json",
			))
		} else {
			rootSeen = true
			rootSourceDigest = baseutils.DigestBytes(raw)
		}
		if err := add(locator, raw); err != nil {
			return nil, "", err
		}
	}
	if !rootSeen {
		return nil, "", fmt.Errorf(
			"%w: transfer payload does not contain root definition %q",
			spec.ErrInvalidRequest,
			payload.RootDefinitionDigest,
		)
	}

	for _, asset := range payload.Assets {
		if err := validate.ValidateAssetManifestEntry(asset.Manifest); err != nil {
			return nil, "", err
		}
		if int64(len(asset.Content)) != asset.Manifest.SizeBytes ||
			baseutils.DigestBytes(asset.Content) != asset.Manifest.Digest {
			return nil, "", fmt.Errorf(
				"%w: transfer asset %q does not match its manifest",
				spec.ErrDigestMismatch,
				asset.Manifest.Path,
			)
		}
		locator := spec.SourceLocator(path.Join(
			destinationDir,
			string(asset.Manifest.Path),
		))
		if err := add(locator, asset.Content); err != nil {
			return nil, "", err
		}
	}

	out := make([]fsDefinitionFile, 0, len(files))
	for _, file := range files {
		out = append(out, file)
	}
	sort.Slice(out, func(left, right int) bool {
		return out[left].Locator < out[right].Locator
	})
	return out, rootSourceDigest, nil
}

func discardFSMaterializedFiles(
	ctx context.Context,
	tool *fstool.FSTool,
	config spec.FSDirectorySourceConfig,
	files []fsMaterializedFile,
) error {
	var discardErrors []error
	for _, file := range slices.Backward(files) {

		stat, err := tool.StatPath(ctx, fstool.StatPathArgs{Path: string(file.Locator)})
		if err != nil {
			discardErrors = append(discardErrors, err)
			continue
		}
		if stat == nil || !stat.Exists {
			continue
		}
		content, err := readFSToolBinary(ctx, tool, string(file.Locator))
		if err != nil {
			discardErrors = append(discardErrors, err)
			continue
		}
		if baseutils.DigestBytes(content) != file.Digest {
			discardErrors = append(discardErrors, fmt.Errorf(
				"%w: materialized file %q changed before compensation",
				spec.ErrConflict,
				file.Locator,
			))
			continue
		}
		_, err = tool.DeleteFile(ctx, fstool.DeleteFileArgs{
			Path:     string(file.Locator),
			TrashDir: filepath.Join(config.RootPath, spec.ManagedArtifactTrashDirectoryName),
		})
		if err != nil {
			discardErrors = append(discardErrors, err)
		}
	}
	return errors.Join(discardErrors...)
}

func fsToolForManagedSource(
	source spec.ArtifactSource,
) (*fstool.FSTool, spec.FSDirectorySourceConfig, spec.Digest, error) {
	canonicalConfig, err := baseutils.CanonicalizeJSON(source.Config)
	if err != nil {
		return nil, spec.FSDirectorySourceConfig{}, "", err
	}
	decoder := json.NewDecoder(bytes.NewReader(canonicalConfig))
	decoder.DisallowUnknownFields()
	var config spec.FSDirectorySourceConfig
	if err := decoder.Decode(&config); err != nil {
		return nil, spec.FSDirectorySourceConfig{}, "", err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return nil, spec.FSDirectorySourceConfig{}, "", errors.New(
				"filesystem source config contains trailing JSON",
			)
		}
		return nil, spec.FSDirectorySourceConfig{}, "", err
	}
	if err := validate.ValidateFSDirectorySourceConfig(config); err != nil {
		return nil, spec.FSDirectorySourceConfig{}, "", err
	}
	if !config.ManagedByApp {
		return nil, spec.FSDirectorySourceConfig{}, "", fmt.Errorf(
			"%w: filesystem source %q is not managed by the application",
			spec.ErrUnsupported,
			source.SourceID,
		)
	}
	tool, err := fstool.NewFSTool(
		fstool.WithAllowedRoots([]string{config.RootPath}),
		fstool.WithWorkBaseDir(config.RootPath),
		fstool.WithBlockSymlinks(true),
	)
	if err != nil {
		return nil, spec.FSDirectorySourceConfig{}, "", err
	}
	return tool, config, baseutils.DigestBytes(canonicalConfig), nil
}

func readFSToolBinary(
	ctx context.Context,
	tool *fstool.FSTool,
	filePath string,
) ([]byte, error) {
	outputs, err := tool.ReadFile(ctx, fstool.ReadFileArgs{
		Path:     filePath,
		Encoding: binaryEncoding,
	})
	if err != nil {
		return nil, err
	}
	for _, output := range outputs {
		switch output.Kind {
		case llmtoolsSpec.ToolOutputKindFile:
			if output.FileItem != nil {
				return base64.StdEncoding.DecodeString(output.FileItem.FileData)
			}
		case llmtoolsSpec.ToolOutputKindImage:
			if output.ImageItem != nil {
				return base64.StdEncoding.DecodeString(output.ImageItem.ImageData)
			}
		case llmtoolsSpec.ToolOutputKindText:
			if output.TextItem != nil {
				return []byte(output.TextItem.Text), nil
			}
		default:
		}
	}
	return nil, errors.New("filesystem read returned no readable content")
}

func encodeFSDefinitionReceipt(receipt fsDefinitionReceipt) (string, error) {
	raw, err := json.Marshal(receipt)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func decodeFSDefinitionReceipt(value string) (fsDefinitionReceipt, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > spec.MaxTransferReceiptBytes*2 {
		return fsDefinitionReceipt{}, fmt.Errorf(
			"%w: filesystem definition receipt size is invalid",
			spec.ErrInvalidRequest,
		)
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return fsDefinitionReceipt{}, fmt.Errorf("decode filesystem definition receipt: %w", err)
	}
	if len(raw) > spec.MaxTransferReceiptBytes {
		return fsDefinitionReceipt{}, fmt.Errorf(
			"%w: filesystem definition receipt is too large",
			spec.ErrInvalidRequest,
		)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var receipt fsDefinitionReceipt
	if err := decoder.Decode(&receipt); err != nil {
		return fsDefinitionReceipt{}, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return fsDefinitionReceipt{}, errors.New("definition receipt contains trailing JSON")
		}
		return fsDefinitionReceipt{}, err
	}
	if receipt.Format != fsDefinitionReceiptFormat ||
		receipt.SourceID == "" ||
		receipt.ConfigDigest == "" ||
		len(receipt.Files) == 0 ||
		len(receipt.Files) > spec.MaxTransferMaterializedFiles {
		return fsDefinitionReceipt{}, fmt.Errorf(
			"%w: malformed filesystem definition receipt",
			spec.ErrInvalidRequest,
		)
	}
	if err := validate.ValidateDigest(receipt.ConfigDigest); err != nil {
		return fsDefinitionReceipt{}, fmt.Errorf("%w: receipt config digest: %w", spec.ErrInvalidRequest, err)
	}
	seen := make(map[spec.SourceLocator]struct{}, len(receipt.Files))
	for _, file := range receipt.Files {
		if err := validate.ValidateCatalogResourceKey(spec.CatalogResourceKey{
			SourceID: receipt.SourceID,
			Locator:  file.Locator,
		}); err != nil {
			return fsDefinitionReceipt{}, fmt.Errorf("%w: receipt file: %w", spec.ErrInvalidRequest, err)
		}
		if err := validate.ValidateDigest(file.Digest); err != nil {
			return fsDefinitionReceipt{}, fmt.Errorf("%w: receipt file digest: %w", spec.ErrInvalidRequest, err)
		}
		if _, duplicate := seen[file.Locator]; duplicate {
			return fsDefinitionReceipt{}, fmt.Errorf(
				"%w: duplicate receipt locator %q",
				spec.ErrInvalidRequest,
				file.Locator,
			)
		}
		seen[file.Locator] = struct{}{}
	}
	return receipt, nil
}

var _ spec.DefinitionMaterializer = (*FSDefinitionMaterializer)(nil)

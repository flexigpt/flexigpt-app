package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// # Preview which files would change:
//     go run ./cmd/scripts/rename-model-aliases -root .

// # Apply changes:
//     go run ./cmd/scripts/rename-model-aliases -root . -write

// # Review and validate:
//     git diff
//     go test ./...

// # Limit it to your target directory:
//     go run ./cmd/scripts/rename-model-aliases \
//       -root ./internal/artifactory-go \
//       -write

const (
	internalPrefix = "github.com/flexigpt/flexigpt-app/internal/"
	prefix         = internalPrefix + "artifactory-go/store/"
)

// Exact import path -> required alias.
// Applies to both unaliased imports and any existing explicit alias.
var targets = map[string]string{
	prefix + "secret/model":            "secretModel",
	prefix + "overlay/model":           "overlayModel",
	prefix + "artifact/model":          "artifactModel",
	prefix + "root/model":              "rootModel",
	prefix + "source/model":            "sourceModel",
	prefix + "artifact/catalog/model":  "catalogModel",
	prefix + "definition/model":        "definitionModel",
	prefix + "definition/schema/model": "schemaModel",
	prefix + "flow/resource/model":     "resourceModel",
	prefix + "flow/refresh/model":      "refreshModel",
	prefix + "flow/install/model":      "installModel",

	prefix + "flow/artifactcleanup": "artifactcleanupFlow",
	prefix + "flow/install":         "installFlow",
	prefix + "flow/managepackage":   "managepackageFlow",
	prefix + "flow/refresh":         "refreshFlow",
	prefix + "flow/resource":        "resourceFlow",
	prefix + "secret/impl":          "secretimpl",

	internalPrefix + "artifactcontract/topology": "documentTopology",
	internalPrefix + "conversation/spec":         "conversationSpec",
	internalPrefix + "inferencewrapper/spec":     "inferencewrapperSpec",
	internalPrefix + "setting/spec":              "settingSpec",
	internalPrefix + "workspace/store/domain":    "workspaceDomain",

	"github.com/modelcontextprotocol/go-sdk/auth":         "mcpSDKAuth",
	"github.com/modelcontextprotocol/go-sdk/auth/extauth": "mcpSDKExtAuth",
}

type packageNameResolver struct {
	dir   string
	names map[string]string
}

type change struct {
	path string
	data []byte
	mode fs.FileMode
}

func main() {
	root := flag.String("root", ".", "directory to scan recursively")
	write := flag.Bool("write", false, "write changes; otherwise dry run")
	flag.Parse()

	resolver := &packageNameResolver{
		dir:   *root,
		names: make(map[string]string),
	}

	var changes []change

	// Prepare every change before writing any files.
	err := filepath.WalkDir(*root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}

		// Do not follow symlinks or process non-Go files.
		if !entry.Type().IsRegular() || filepath.Ext(path) != ".go" {
			return nil
		}

		data, err := rewrite(path, resolver)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if data == nil {
			return nil // Nothing to rename.
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		changes = append(changes, change{
			path: path,
			data: data,
			mode: info.Mode().Perm(),
		})
		return nil
	})
	if err != nil {
		fail(err)
	}

	for _, c := range changes {
		if *write {
			if err := os.WriteFile(c.path, c.data, c.mode); err != nil {
				fail(fmt.Errorf("%s: %w", c.path, err))
			}
			fmt.Println("Updated:", c.path)
		} else {
			fmt.Println("Would update:", c.path)
		}
	}

	if *write {
		fmt.Printf("Updated %d file(s).\n", len(changes))
	} else {
		fmt.Printf(
			"Dry run: %d file(s). Run again with -write to apply.\n",
			len(changes),
		)
	}
}

// rewrite returns nil when the file needs no changes.
func rewrite(path string, resolver *packageNameResolver) ([]byte, error) {
	fset := token.NewFileSet()

	// Keep object resolution enabled: Ident.Obj helps distinguish
	// local variables from imported package names.
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	renames := make(map[string]string)
	matchedImports := make(map[*ast.ImportSpec]string)

	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}

		newAlias, wanted := targets[importPath]
		if !wanted {
			continue
		}

		var oldAlias string
		if imp.Name != nil {
			oldAlias = imp.Name.Name
			// Keep side-effect and dot imports unchanged.
			if oldAlias == "_" || oldAlias == "." {
				continue
			}
			if oldAlias == newAlias {
				continue // Already explicitly aliased correctly.
			}
		} else {
			// The declared package name need not match the path's last segment.
			oldAlias, err = resolver.name(importPath)
			if err != nil {
				return nil, err
			}
		}

		matchedImports[imp] = newAlias
		if oldAlias == newAlias {
			// Only the explicit alias is missing; existing uses stay unchanged.
			continue
		}

		if previous, exists := renames[oldAlias]; exists && previous != newAlias {
			return nil, fmt.Errorf(
				"ambiguous import name %q: requested both %q and %q",
				oldAlias, previous, newAlias,
			)
		}
		renames[oldAlias] = newAlias
	}

	if len(matchedImports) == 0 {
		return nil, nil
	}

	// Only reject real file-scope collisions. Do not reject function-local
	// variables, parameters, struct fields, or selector fields which may
	// legitimately have the same name as a package import alias.
	if err := checkImportAliasCollisions(file, matchedImports, resolver); err != nil {
		return nil, err
	}

	// Rename package qualifiers.
	//
	//    root.Type       -> rootModel.Type
	//    root.Type.Field -> rootModel.Type.Field
	//    obj.root.Type   -> unchanged
	//
	// Locally declared variables named root/artifact/etc. are skipped.
	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		id, ok := selector.X.(*ast.Ident)
		if !ok || id.Obj != nil {
			return true
		}

		if newAlias, found := renames[id.Name]; found {
			id.Name = newAlias
		}

		return true
	})

	// Add or update the explicit aliases after rewriting their uses.
	for imp, newAlias := range matchedImports {
		if imp.Name == nil {
			imp.Name = &ast.Ident{NamePos: imp.Path.Pos(), Name: newAlias}
		} else {
			imp.Name.Name = newAlias
		}
	}

	var output bytes.Buffer
	if err := format.Node(&output, fset, file); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

// checkImportAliasCollisions verifies that the aliases introduced by this
// rewrite do not collide with another import or a package-level declaration.
//
// It intentionally does not reject local variables, function parameters,
// struct fields, methods, or selector fields with the same name. Those are in
// nested scopes and are not automatically an import collision.
func checkImportAliasCollisions(
	file *ast.File,
	matchedImports map[*ast.ImportSpec]string,
	resolver *packageNameResolver,
) error {
	requestedAliases := make(map[string]bool)
	for _, newAlias := range matchedImports {
		requestedAliases[newAlias] = true
	}

	// Determine the final alias of every import after this rewrite. Only check
	// aliases that this script is introducing or changing.
	seenAliases := make(map[string]string)
	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return err
		}

		alias, err := importAliasAfterRewrite(imp, importPath, matchedImports, resolver)
		if err != nil {
			return err
		}

		// Blank and dot imports do not introduce a normal package alias.
		if alias == "_" || alias == "." || !requestedAliases[alias] {
			continue
		}

		if previousPath, exists := seenAliases[alias]; exists {
			return fmt.Errorf(
				"import alias %q would be used by both %q and %q",
				alias,
				previousPath,
				importPath,
			)
		}

		seenAliases[alias] = importPath
	}

	// A package-level declaration shares the file block with imports and is a
	// real collision. For example:
	//
	//     var mcpAuth = ...
	//
	// cannot coexist with:
	//
	//     mcpAuth "github.com/modelcontextprotocol/go-sdk/auth"
	for alias := range requestedAliases {
		if packageDeclaresName(file, alias) {
			return fmt.Errorf(
				"new alias %q conflicts with a package-level declaration; review manually",
				alias,
			)
		}
	}

	return nil
}

// importAliasAfterRewrite returns the alias that an import will have after the
// pending rewrite. For unaliased imports, it resolves the package clause using
// "go list", because the package name may differ from the path's last segment.
func importAliasAfterRewrite(
	imp *ast.ImportSpec,
	importPath string,
	matchedImports map[*ast.ImportSpec]string,
	resolver *packageNameResolver,
) (string, error) {
	if newAlias, rewritten := matchedImports[imp]; rewritten {
		return newAlias, nil
	}

	if imp.Name != nil {
		return imp.Name.Name, nil
	}

	// Cgo's special import cannot be resolved through "go list".
	if importPath == "C" {
		return "C", nil
	}

	return resolver.name(importPath)
}

// packageDeclaresName reports whether name is declared in the package/file
// scope. Local declarations are intentionally ignored.
func packageDeclaresName(file *ast.File, name string) bool {
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			// Methods belong to a receiver type, not package scope.
			if decl.Recv == nil && decl.Name != nil && decl.Name.Name == name {
				return true
			}

		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					if spec.Name != nil && spec.Name.Name == name {
						return true
					}

				case *ast.ValueSpec:
					for _, ident := range spec.Names {
						if ident != nil && ident.Name == name {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// name resolves the actual package clause, not the last import-path segment.
// Cache lookups so each unaliased target needs at most one "go list" call.
func (r *packageNameResolver) name(importPath string) (string, error) {
	if name, ok := r.names[importPath]; ok {
		return name, nil
	}

	cmd := exec.CommandContext(context.Background(), "go", "list", "-find", "-f", "{{.Name}}", importPath)
	cmd.Dir = r.dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(
			"resolve package name for %q: %w; %s",
			importPath, err, strings.TrimSpace(stderr.String()),
		)
	}

	name := strings.TrimSpace(string(output))
	if name == "_" || !token.IsIdentifier(name) {
		return "", fmt.Errorf(
			"go list returned invalid package name %q for %q",
			name, importPath,
		)
	}

	r.names[importPath] = name
	return name, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}

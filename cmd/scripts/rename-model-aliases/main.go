package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

const prefix = "github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/"

// Exact import path -> existing alias.
// Each alias becomes existingAlias + "Model".
var targets = map[string]string{
	prefix + "artifact/model":         "artifact",
	prefix + "root/model":             "root",
	prefix + "source/model":           "source",
	prefix + "artifact/catalog/model": "catalog",
	prefix + "definition/model":       "definition",
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

		data, err := rewrite(path)
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
func rewrite(path string) ([]byte, error) {
	fset := token.NewFileSet()

	// Keep object resolution enabled: Ident.Obj helps distinguish
	// local variables from imported package names.
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	renames := make(map[string]string)
	var matchedImports []*ast.ImportSpec

	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			return nil, err
		}

		oldAlias, wanted := targets[importPath]
		if !wanted || imp.Name == nil || imp.Name.Name != oldAlias {
			continue
		}

		renames[oldAlias] = oldAlias + "Model"
		matchedImports = append(matchedImports, imp)
	}

	if len(renames) == 0 {
		return nil, nil
	}

	// Conservative collision check. This also catches local declarations
	// that could shadow a newly introduced import alias.
	newNames := make(map[string]bool)
	for _, newAlias := range renames {
		newNames[newAlias] = true
	}

	var conflict string
	ast.Inspect(file, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && newNames[id.Name] {
			conflict = id.Name
		}
		return true
	})

	if conflict != "" {
		return nil, fmt.Errorf(
			"new alias %q already appears as an identifier; review manually",
			conflict,
		)
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

	// Rename only the imports that matched the exact path/alias pairs.
	for _, imp := range matchedImports {
		imp.Name.Name = renames[imp.Name.Name]
	}

	var output bytes.Buffer
	if err := format.Node(&output, fset, file); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}

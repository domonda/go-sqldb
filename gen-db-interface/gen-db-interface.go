package main

import (
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"

	"github.com/ungerik/go-fs"
	"github.com/vburenin/ifacemaker/maker"
)

func main() {
	var implPackageDir fs.File
	switch len(os.Args) {
	case 1:
		implPackageDir = fs.CurrentWorkingDir()
	case 2:
		implPackageDir = fs.File(os.Args[1])
	default:
		log.Fatalf("need 0 or 1 arguments but got %d", len(os.Args))
	}
	if !implPackageDir.IsDir() {
		log.Fatalf("not a package directory: %s", string(implPackageDir))
	}

	var implPackageFiles []string
	err := implPackageDir.ListDir(func(file fs.File) error {
		if !file.IsDir() && file.Ext() == ".go" && !strings.HasSuffix(file.Name(), "_test.go") {
			implPackageFiles = append(implPackageFiles, file.LocalPath())
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	packageDir := implPackageDir.Dir()
	packageName := packageNameOfDir(packageDir.LocalPath())
	outputFile := packageDir.Join("database.go")

	source, err := maker.Make(
		implPackageFiles, // Implementation struct filenames
		"Impl",           // Implementation struct name
		"GENERATED",      // Interface file comment
		packageName,      // Interface package
		"Database",       // Interface type mame
		"Database interface for package "+packageName, // Interface type comment
		true,  // Copy docs from methods
		false, // Copy type doc from struct
	)
	if err != nil {
		log.Fatal(err)
	}

	err = outputFile.WriteAll(source)
	if err != nil {
		log.Fatal(err)
	}
}

func packageNameOfDir(packageDir string) string {
	pkgs, err := parser.ParseDir(token.NewFileSet(), packageDir, nil, 0)
	if err != nil {
		log.Fatal(err)
	}
	if len(pkgs) != 1 {
		log.Fatalf("found %d packages in directory %q", len(pkgs), packageDir)
	}
	for name := range pkgs {
		return name
	}
	panic("never reached")
}

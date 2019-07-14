package main

import (
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/vburenin/ifacemaker/maker"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("need implementation filename")
	}

	packageDir := ".."
	packageName := packageNameOfDir(packageDir)

	source, err := maker.Make(
		os.Args[1:2], // Implementation struct filenames
		"Impl",       // Implementation struct name
		"GENERATED",  // Interface file comment
		packageName,  // Interface package
		"Database",   // Interface type mame
		"Database interface for package "+packageName, // Interface type comment
		true,  // Copy docs from methods
		false, // Copy type doc from struct
	)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(filepath.Join(packageDir, "database.go"), source, 0664)
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

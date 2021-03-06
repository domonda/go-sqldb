package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strings"

	"github.com/ungerik/go-astvisit"
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

	// For debugging:
	// implPackageDir = "~/go/src/github.com/domonda/domonda-service/pkg/client/clientdb"

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
		implPackageFiles,                // Implementation struct filenames
		"impl",                          // Implementation struct name
		"GENERATED by gen-db-interface", // Interface file comment
		packageName,                     // Interface package
		"Database",                      // Interface type mame
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

	// Parse written interface and create functions calling on db var

	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, outputFile.Name(), source, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	visitor := &interfaceTypeVisitor{
		interfaceVar: "db",
		fset:         fset,
		source:       source,
		out:          bytes.NewBuffer(source),
	}

	astvisit.Visit(f, visitor, nil)

	err = outputFile.WriteAll(visitor.out.Bytes())
	if err != nil {
		log.Fatal(err)
	}
}

type interfaceTypeVisitor struct {
	astvisit.VisitorImpl

	interfaceVar string
	fset         *token.FileSet
	source       []byte
	out          *bytes.Buffer
}

func (v *interfaceTypeVisitor) VisitInterfaceType(interfaceType *ast.InterfaceType, cursor astvisit.Cursor) bool {
	for _, method := range interfaceType.Methods.List {
		// ast.Print(v.fset, method)

		methodName := method.Names[0].Name
		if methodName == "Conn" || methodName == "Begin" || methodName == "Commit" || methodName == "Rollback" {
			continue
		}

		methodType := method.Type.(*ast.FuncType)
		methodSignature := v.source[v.fset.Position(method.Pos()).Offset:v.fset.Position(method.End()).Offset]

		if method.Doc != nil {
			for _, comment := range method.Doc.List {
				fmt.Fprintf(v.out, "\n%s", comment.Text)
			}
		}

		fmt.Fprintf(v.out, "\nfunc %s {\n\t", methodSignature)
		if methodType.Results.NumFields() > 0 {
			fmt.Fprint(v.out, "return ")
		}

		fmt.Fprintf(v.out, "%s.%s(", v.interfaceVar, methodName)
		first := true
		for _, param := range methodType.Params.List {
			for _, name := range param.Names {
				if first {
					first = false
				} else {
					fmt.Fprint(v.out, ", ")
				}
				fmt.Fprint(v.out, name)
			}
		}
		fmt.Fprint(v.out, ")\n}\n")
	}
	return true
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

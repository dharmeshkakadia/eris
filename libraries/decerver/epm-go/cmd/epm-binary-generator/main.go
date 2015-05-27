package main

import (
	"fmt"
	"github.com/eris-ltd/epm-go/utils"
	"go/ast"
	gofmt "go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

// eg. `epm-binary-generator cmdSourcePath commandsPkg chain1 chain2 chain3 ...`

var (
	fset = token.NewFileSet() // positions are relative to fset
)

func main() {
	args := os.Args
	if len(args) < 4 {
		fmt.Println("Please enter a path to the executables source, a path to the modules file, and a list of chains to compile against")
		os.Exit(1)
	}

	cmdPath := args[1]                        // path to executables source
	commandsPkg, err := filepath.Abs(args[2]) // library with epm-binary-generator tags
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	chains := args[3:] // list of chains to compile against
	_, _ = cmdPath, chains

	// find all imports
	chainDecl, imports, err := parseImports(commandsPkg)
	if err != nil {
		fmt.Println("Error parsing package", commandsPkg, err)
		os.Exit(1)
	}

	// we need to back up the files we'll be overwriting
	// imports and files are bijective
	backupFolder := path.Join(utils.ErisLtd, "epm-go", "commands", ".backup")
	if _, err := os.Stat(backupFolder); err != nil {
		if err := os.Mkdir(backupFolder, 0700); err != nil {
			fmt.Println("Error making backup directory", err)
			os.Exit(1)
		}
		for _, imp := range imports {
			fmt.Println("backing up ", imp.fileName)
			err := utils.Copy(path.Join(commandsPkg, imp.fileName), path.Join(backupFolder, imp.fileName))
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	err = installChains(commandsPkg, chainDecl, imports, chains)
	if err != nil {
		fmt.Println("Error on chain install:", err)
	}
}

func installChains(commandsPkg string, chainDecl *ast.BasicLit, imports []*Import, chains []string) error {
	cur, _ := os.Getwd()
	os.Chdir(path.Join(utils.ErisLtd, "epm-go"))
	defer os.Chdir(cur)

	// for each chain, change the import statements, write to file, install
	for _, chain := range chains {
		for _, imp := range imports {
			// update the import path
			imp.importSpec.Path.Value = stringImport(chain)

			// update the chain var
			chainDecl.Value = "\"" + chain + "\""

			// write to file
			f, err := os.Create(path.Join(commandsPkg, imp.fileName))
			if err != nil {
				return fmt.Errorf(fmt.Sprintln("Could not create file", imp.fileName, err))
			}
			gofmt.Node(f, fset, imp.file)
			f.Close()

		}

		// install the binary locally and move to go bin
		cmd := exec.Command("go", "build", "-o", "epm-"+chain, "./cmd/epm")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf(fmt.Sprintln("Error building", chain, err))
		}
		err = os.Rename("epm-"+chain, path.Join(utils.GoPath, "bin", "epm-"+chain))
		if err != nil {
			return fmt.Errorf(fmt.Sprintln("Error moving binary to GoBin", err))
		}
	}

	// now clean up by reseting to thelonious imports
	// and setting CHAIN to ""
	for _, imp := range imports {
		// update the import path
		imp.importSpec.Path.Value = stringImport("thelonious")

		// update the chain var
		chainDecl.Value = "\"\""

		// write to file
		f, err := os.Create(path.Join(commandsPkg, imp.fileName))
		if err != nil {
			return fmt.Errorf(fmt.Sprintln("Could not create file", imp.fileName, err))
		}
		gofmt.Node(f, fset, imp.file)
		f.Close()
	}

	// finally, install plain ole epm
	cmd := exec.Command("go", "install", "./cmd/epm")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Error installing plain epm %v", err)
	}

	return nil
}

func stringImport(chain string) string {
	return "\"" + path.Join("github.com", "eris-ltd", "epm-go", "commands", "modules", chain) + "\""
}

// an import is an epm-binary-generator tag followed by a pkg import
// we assume only one import per file
type Import struct {
	fileName   string
	file       *ast.File
	comment    *ast.Comment
	importSpec *ast.ImportSpec
}

// parse for the import indicators in the pkg
func parseImports(commandsPkg string) (*ast.BasicLit, []*Import, error) {
	// for each file in the dir, parse, check for tag
	fs, err := ioutil.ReadDir(commandsPkg)
	if err != nil {
		return nil, nil, err
	}

	validFiles := []string{}
	for _, f := range fs {
		name := f.Name()
		if !f.IsDir() && path.Ext(name) == ".go" && !strings.HasSuffix(name, "_test.go") {
			validFiles = append(validFiles, name)
		}
	}

	var chainDecl *ast.BasicLit
	imports := []*Import{}
	for _, f := range validFiles {
		buf, err := ioutil.ReadFile(path.Join(commandsPkg, f))
		if err != nil {
			return nil, nil, err
		}
		astFile, err := parser.ParseFile(fset, "", buf, parser.ParseComments)
		if err != nil {
			return nil, nil, fmt.Errorf(fmt.Sprintln("Error parsing modules file:", err))
		}

		// find the CHAIN variable declaration
		for _, decl := range astFile.Decls {
			if gendecl, ok := decl.(*ast.GenDecl); ok {
				if gendecl.Tok == token.CONST {
					if valspec, ok := gendecl.Specs[0].(*ast.ValueSpec); ok {
						names := valspec.Names
						if len(names) == 1 && names[0].Name == "CHAIN" {
							val := valspec.Values[0]
							if vallit, ok := val.(*ast.BasicLit); ok {
								if chainDecl != nil {
									return nil, nil, fmt.Errorf("const CHAIN should only be declared once!")
								}
								chainDecl = vallit
							}
						}
					}
				}

			}

		}

		for _, c := range astFile.Comments {
			for _, cc := range c.List {
				if strings.HasPrefix(cc.Text[2:], "epm-binary-generator:") {
					txt := cc.Text[2+len("epm-binary-generator:"):]
					if strings.HasPrefix(txt, "IMPORT") {
						// the next line is our import
						// loop through the imports and find the one after this comment
						// in properly formatted go, they differ by 2
						for _, imp := range astFile.Imports {
							if imp.Pos()-cc.End() == 2 {
								imports = append(imports, &Import{
									fileName:   f,
									file:       astFile,
									comment:    cc,
									importSpec: imp,
								})
							}
						}

					}
				}
			}
		}
	}
	return chainDecl, imports, nil
}

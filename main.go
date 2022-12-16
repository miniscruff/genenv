package main

import (
	"bytes"
	"fmt"
	"go/doc"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"

	flag "github.com/spf13/pflag"
)

type PackageTypes struct {
	DocTypes map[string]*doc.Type
	Imports  map[string]string
}

var logLine func(args ...any)

func main() {
	var (
		pkgName    string
		fileDir    string
		configType string
		genFile    string
		verbose    bool
		envFile    string
	)

	flag.StringVarP(&pkgName, "package", "p", "", "Name of config type, defaults to dir")
	flag.StringVarP(&fileDir, "dir", "d", "", "Directory of our config file")
	flag.StringVarP(&configType, "config", "c", "", "Name of config type")
	flag.StringVarP(&genFile, "file", "f", "", "Name of generated file to write to")
	flag.BoolVarP(&verbose, "verbose", "v", false, "verbose logging")
	flag.StringVarP(&envFile, "env", "e", "", "Name of file to write env example to")

	flag.Parse()

	if pkgName == "" {
		pkgName = fileDir
	}

	cfg := GenConfig{
		PackageName:   pkgName,
		FileDir:       fileDir,
		ConfigType:    configType,
		GoOutputFile:  genFile,
		EnvOutputFile: envFile,
		Verbose:       verbose,
	}
	if err := GenEnv(cfg); err != nil {
		log.Fatal(err)
	}
}

type GenConfig struct {
	PackageName   string
	FileDir       string
	ConfigType    string
	GoOutputFile  string
	EnvOutputFile string
	Verbose       bool
}

func GenEnv(cfg GenConfig) error {
	if cfg.Verbose {
		logLine = func(args ...any) {
			fmt.Println(args...)
		}
	} else {
		logLine = func(args ...any) {
		}
	}

	fset, pkgTypes, err := loadDocPackage(cfg.FileDir, cfg.PackageName)
	if err != nil {
		return err
	}

	imports := &ImportCache{}
	errs := &ErrorCache{}
	parsers := &ParserCache{}
	queue := &QueueCache{}

	// initial states
	imports.Add("os", "os")

	queue.Add(cfg.ConfigType)

	var w bytes.Buffer

	for !queue.IsEmpty() {
		firstType := queue.Pop()

		tpe, found := pkgTypes.DocTypes[firstType]
		if !found {
			return fmt.Errorf("config type '%v' not found", firstType)
		}

		b, err := NewStructBuilder(
			tpe,
			pkgTypes,
			pkgTypes.Imports,
			cfg.ConfigType,
			queue,
			parsers,
			errs,
			imports,
		)
		if err != nil {
			return err
		}
		b.Write(&w)
	}

	// write parsers to W so it can add imports and errors
	if err := parsers.Write(&w); err != nil {
		return err
	}

	// now write the file in order:
	// package -> imports -> errors -> configs + parsers -> newline
	var topWriter bytes.Buffer
	writeF(&topWriter, "package %v\n\n", cfg.PackageName)
	imports.Write(&topWriter)
	errs.Write(&topWriter)
	topWriter.Write(w.Bytes())
	writeF(&topWriter, "\n")

	formattedBytes, err := format.Source(topWriter.Bytes())
	if err != nil {
		// Useful for debugging
		os.Stderr.Write(topWriter.Bytes())
		return fmt.Errorf("error formatting: %w", err)
	}

	outputFile := cfg.GoOutputFile
	if outputFile == "" {
		tokFile := fset.File(pkgTypes.DocTypes[cfg.ConfigType].Decl.TokPos)
		nameNoExt := strings.TrimSuffix(tokFile.Name(), ".go")
		outputFile = nameNoExt + "_gen.go"
	}

	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}

	f.Write(formattedBytes)
	return nil
}

func loadDocPackage(dirName, pkgName string) (*token.FileSet, *PackageTypes, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "./"+dirName, func(fi fs.FileInfo) bool {
		logLine("file found:", fi.Name())
		return true
	}, parser.ParseComments)
	if err != nil {
		return fset, nil, err
	}

	for name := range pkgs {
		logLine("package found:", name)
	}

	pkg, found := pkgs[pkgName]
	if !found {
		return fset, nil, fmt.Errorf("package '%v' not found", pkgName)
	}

	docPkg := doc.New(pkg, "./", 0)
	pkgTypes := &PackageTypes{
		Imports:  make(map[string]string),
		DocTypes: make(map[string]*doc.Type),
	}

	for _, files := range pkg.Files {
		for _, fileImp := range files.Imports {
			importPath := strings.Trim(fileImp.Path.Value, "\"")
			split := strings.Split(importPath, "/")
			importKey := strings.Trim(split[len(split)-1], " ")
			pkgTypes.Imports[importKey] = importPath
			logLine("import: [", importKey, "] =", importPath)
		}
	}
	// fmt.Println(pkgTypes.Imports)

	for _, t := range docPkg.Types {
		pkgTypes.DocTypes[t.Name] = t
	}

	return fset, pkgTypes, nil
}

func writeF(w io.Writer, format string, args ...any) error {
	_, err := w.Write([]byte(fmt.Sprintf(format, args...)))
	return err
}

package main

import (
	"fmt"
	"go/ast"
	"io"
	"reflect"
	"strings"
)

type Field struct {
	varName       string
	typeName      string
	docs          string
	defaultValue  string
	envKey        string
	required      bool
	slice         bool
	customType    bool
	rootTypeField bool
	buildType     string
	hasTypeField bool

	imports map[string]string

	queue       *QueueCache
	parsers     *ParserCache
	errs        *ErrorCache
	importCache *ImportCache
}

func NewField(
	field *ast.Field,
	pkgTypes *PackageTypes,
	imports map[string]string,
	rootType bool,
	queue *QueueCache,
	parsers *ParserCache,
	errs *ErrorCache,
	importCache *ImportCache,
) (*Field, error) {
	f := &Field{
		defaultValue:  "", // default should be empty
		required:      true,
		slice:         false,
		docs:          field.Doc.Text(),
		imports:       imports,
		rootTypeField: rootType,
		queue:         queue,
		parsers:       parsers,
		errs:          errs,
		importCache:   importCache,
	}

	switch fieldType := field.Type.(type) {
	case *ast.Ident:
		f.typeName = fieldType.Name
		if _, found := pkgTypes.DocTypes[f.typeName]; found {
			f.customType = true
		}
	case *ast.ArrayType:
		rootType := fieldType.Elt.(*ast.Ident)
		f.typeName = rootType.Name
		f.slice = true
	case *ast.StarExpr:
		rootType := fieldType.X.(*ast.Ident)
		f.typeName = rootType.Name
		f.required = false
		if _, found := pkgTypes.DocTypes[f.typeName]; found {
			f.customType = true
		}
	case *ast.SelectorExpr:
		rootType := fieldType.Sel
		f.typeName = fmt.Sprintf("%v.%v", fieldType.X, rootType.Name)
		logLine("field type:", f.typeName)
	default:
		return f, fmt.Errorf("unknown field type: %T for field: '%v'", fieldType, field.Names[0])
	}

	// this checks for nameless variables that inherit the type name
	if len(field.Names) > 0 {
		f.varName = field.Names[0].Name
	} else {
		f.varName = f.typeName
	}

	f.envKey = varNameToKey(f.varName)

	if field.Tag != nil {
		tags := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		if def, ok := tags.Lookup("default"); ok {
			f.required = false
			f.defaultValue = fmt.Sprintf("\"%v\"", def)
		}

		if env, ok := tags.Lookup("env"); ok {
			f.envKey = env
		}

		if bType, ok := tags.Lookup("buildType"); ok {
			f.buildType = bType
		}
	}

	return f, nil
}

func (f *Field) Write(w io.Writer) error {
	envKey := fmt.Sprintf("\"%v\"", f.envKey)
	if !f.rootTypeField {
		envKey = fmt.Sprintf("prefix + \"_%v\"", f.envKey)
	}

	// if we are part of a conditional type, wrap our new in an if
	// making sure we are actually using the type
	if f.hasTypeField {
		writeF(
			w,
			"if c.Type == \"%v\" {\n",
			f.envKey,
		)
	}

	if f.customType {
		f.queue.Add(f.typeName)
		writeF(
			w,
			"c.%v, err = New%v(%v)\nif err != nil {\n return c, err\n}",
			f.varName,
			f.typeName,
			envKey,
		)

	} else {
		parserType := f.getParserFunc()

		parseArgs := fmt.Sprintf("%v, %v", f.defaultValue, envKey)
		if f.required {
			parseArgs = envKey
		}

		writeF(
			w,
			"c.%s, err = %v(%v)\nif err != nil {\nreturn c, err\n}",
			f.varName,
			parserType,
			parseArgs,
		)
	}

	if f.hasTypeField {
		writeF(
			w,
			"\n}",
		)
	}

	writeF(w, "\n\n")

	return nil
}

func (f *Field) getParserFunc() string {
	parser := Parser{
		ReturnType:  f.typeName,
		IsSlice:     f.slice,
		IsRequired:  f.required,
		Errs:        f.errs,
		Imports:     f.imports,
		ImportCache: f.importCache,
	}
	f.parsers.Add(parser.FuncName(), parser)

	return parser.FuncName()
}

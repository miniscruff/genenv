package main

import (
	"go/ast"
	"go/doc"
	"io"
	"strings"
)

type StructBuilder struct {
	pkgTypes  PackageTypes
	us        *doc.Type
	name      string
	rootType  bool
	buildType string

	queue   *QueueCache
	parsers *ParserCache
	errs    *ErrorCache
	imports *ImportCache

	fields map[string]*Field
}

func NewStructBuilder(
	tpe *doc.Type,
	pkgTypes PackageTypes,
	rootTypeName string,
	queue *QueueCache,
	parsers *ParserCache,
	errs *ErrorCache,
	imports *ImportCache,
) (*StructBuilder, error) {
	b := &StructBuilder{
		pkgTypes: pkgTypes,
		us:       tpe,
		rootType: tpe.Name == rootTypeName,
		name:     tpe.Name,
		queue:    queue,
		parsers:  parsers,
		errs:     errs,
		imports:  imports,
		fields:   make(map[string]*Field),
	}

	for _, spec := range b.us.Decl.Specs {
		typeSpec, _ := spec.(*ast.TypeSpec)
		structType, ok := typeSpec.Type.(*ast.StructType)

		if !ok {
			continue
		}

		for _, field := range structType.Fields.List {
			newField, err := NewField(field, b.pkgTypes, b.rootType, b.queue, b.parsers, b.errs, b.imports)
			if err != nil {
				return nil, err
			}

			b.fields[newField.varName] = newField

			logLine("var name:", newField.varName)
			if newField.buildType != "" {
				logLine("found build type:", newField.buildType)
				b.buildType = newField.buildType
			}
		}
	}

	return b, nil
}

func (b *StructBuilder) Write(w io.Writer) error {
	if b.rootType {
		writeF(w,
			"func New%v() (*%v, error) {\n",
			b.name,
			b.name,
		)
	} else {
		writeF(w,
			"func New%v(prefix string) (*%v, error) {\n",
			b.name,
			b.name,
		)
	}

	writeF(w,
		"var (\nerr error\nc *%v\n)\n\n",
		b.name,
	)

	for _, f := range b.fields {
		err := f.Write(w)
		if err != nil {
			return err
		}
	}

	writeF(w, "\nreturn c, err\n}\n\n")

	if b.buildType != "" {
		logLine("using build type:", b.buildType)
		writeF(
			w,
			"func (c *%v) Build() (%v, error) {\n",
			b.name,
			b.buildType,
		)

		writeF(
			w,
			"switch c.Type {\n",
		)

		for n, f := range b.fields {
			if n == "Type" {
				continue
			}

			newFuncName := strings.TrimSuffix(n, "Config")

			writeF(
				w,
				"case \"%v\":\nreturn c.New%v()\n",
				f.envKey,
				newFuncName,
			)
		}

		b.errs.Add("ErrInvalidBuildType", ErrorDef{
			VarName: "ErrInvalidBuildType",
			Desc:    "invalid build type",
		})
		writeF(
			w,
			"default:\nreturn nil, fmt.Errorf(\"%%w: %%v\", %v, c.Type)\n}\n}\n\n",
			"ErrInvalidBuildType",
		)
	}

	return nil
}

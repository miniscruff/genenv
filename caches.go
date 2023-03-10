package main

import (
	"fmt"
	"io"
	"strings"
	"unicode"
)

type (
	Cacher[T any] struct {
		values map[string]T
	}

	Import string
	Parser struct {
		ReturnType  string
		IsSlice     bool
		IsRequired  bool
		Imports     map[string]string
		Errs        *ErrorCache
		ImportCache *ImportCache
	}

	ErrorDef struct {
		VarName string
		Desc    string
	}

	ImportCache struct {
		Cacher[string]
	}

	ParserCache struct {
		Cacher[Parser]
	}

	ErrorCache struct {
		Cacher[ErrorDef]
	}

	QueueCache struct {
		values []string
		seen   map[string]struct{}
	}
)

func (c *QueueCache) Add(key string) {
	if _, found := c.seen[key]; found {
		return
	}

	if c.seen == nil {
		c.seen = make(map[string]struct{})
	}

	logLine("adding item to queue:", key)
	c.seen[key] = struct{}{}
	c.values = append(c.values, key)
}

func (c *QueueCache) Pop() string {
	firstValue := c.values[0]
	c.values = c.values[1:]
	return firstValue
}

func (c *QueueCache) IsEmpty() bool {
	return len(c.values) == 0
}

func (c *Cacher[T]) Add(key string, value T) {
	if _, found := c.values[key]; found {
		return
	}

	if c.values == nil {
		c.values = make(map[string]T)
	}

	c.values[key] = value
}

func (c *ImportCache) Write(w io.Writer) error {
	err := writeF(w, "import (\n")
	// only check the write error once
	if err != nil {
		return err
	}

	for _, imp := range c.values {
		writeF(w, "\"%v\"\n", imp)
	}

	writeF(w, ")\n\n")
	return nil
}

func (c *ErrorCache) Write(w io.Writer) error {
	err := writeF(w, "var (\n")
	// only check the write error once
	if err != nil {
		return err
	}

	for _, e := range c.values {
		writeF(w, "%v = errors.New(\"%v\")\n", e.VarName, e.Desc)
	}

	writeF(w, ")\n\n")
	return nil
}

func (p Parser) RequiredStr() string {
	if p.IsRequired {
		return "Required"
	}
	return "Optional"
}

func (p Parser) ArgsList() string {
	argsList := "def, key string"
	if p.IsRequired {
		argsList = "key string"
	}
	return argsList
}

func (p Parser) FuncName() string {
	sliceStr := ""
	if p.IsSlice {
		sliceStr = "Slice"
	}

	return fmt.Sprintf(
		"Parse%v%v%v",
		strings.ReplaceAll(strings.Title(p.ReturnType), ".", ""),
		sliceStr,
		p.RequiredStr(),
	)
}

func (p *Parser) Write(w io.Writer) error {
	funcName := p.FuncName()
	if _, found := convMap[p.ReturnType]; !found {
		return fmt.Errorf("unknown type: %v", p.ReturnType)
	}

	for _, imp := range convMap[p.ReturnType].Imports {
		p.ImportCache.Add(imp, imp)
	}
	for _, e := range convMap[p.ReturnType].Errs {
		p.Errs.Add(e.VarName, e)
	}

	writeF(
		w,
		"func %v(%v) (%v, error) {\nv, ok := os.LookupEnv(key)\nif !ok {\n",
		funcName,
		p.ArgsList(),
		p.ReturnType,
	)

	if p.IsRequired {
		p.ImportCache.Add("errors", "errors")
		p.ImportCache.Add("fmt", "fmt")
		p.Errs.Add("keyNotFound", ErrorDef{
			VarName: "ErrKeyNotFound",
			Desc:    "env var key not found",
		})

		writeF(
			w,
			"return %v, fmt.Errorf(\"%%w: %%v\", ErrKeyNotFound, key)",
			convMap[p.ReturnType].DefaultValue,
		)
	} else {
		writeF(
			w,
			"v = def",
		)
	}

	writeF(
		w,
		"\n}\n\n",
	)

	writeF(
		w,
		convMap[p.ReturnType].ConvReturnFormat,
		"v",
	)

	if p.ReturnType != "string" {
		p.ImportCache.Add("strconv", "strconv")
	}

	writeF(w, "\n}\n\n")
	return nil
}

func (c *ParserCache) Write(w io.Writer) error {
	for _, parser := range c.values {
		logLine("parser:", parser.FuncName())
		if err := parser.Write(w); err != nil {
			return err
		}
	}

	return nil
}

type ConvInfo struct {
	DefaultValue     string
	ConvReturnFormat string
	Imports          []string
	Errs             []ErrorDef
}

var convMap = map[string]ConvInfo{
	"string": {
		DefaultValue:     "\"\"",
		ConvReturnFormat: "return %v, nil",
	},
	"int": {
		DefaultValue:     "0",
		ConvReturnFormat: "v64, err := strconv.ParseInt(%v, 10, 64)\nif err != nil {\nreturn 0, err\n}\n\nreturn int(v64), nil",
	},
	"bool": {
		DefaultValue: "false",
		Imports:      []string{"strings", "fmt", "errors"},
		Errs: []ErrorDef{
			{
				VarName: "ErrInvalidBool",
				Desc:    "invalid bool value",
			},
		},
		ConvReturnFormat: `switch strings.ToLower(%v) {
		case "y", "yes", "true", "t", "1", "on":
			return true, nil
		case "n", "no", "false", "f", "0", "off":
			return false, nil
		default:
			return false, fmt.Errorf("%%w: %%v", ErrInvalidBool, v)
		}`,
	},
	"time.Duration": {
		DefaultValue: "0",
		Imports:      []string{"time"},
		ConvReturnFormat: `vd, err := time.ParseDuration(%v)
		if err != nil {
			return 0, err
		}

		return vd, nil`,
	},
}

func varNameToKey(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if sb.Len() > 0 && unicode.IsUpper(r) {
			sb.WriteString("_")
		}
		sb.WriteRune(unicode.ToUpper(r))
	}

	return sb.String()
}

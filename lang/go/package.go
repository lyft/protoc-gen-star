package pgsgo

import (
	"fmt"
	"go/token"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	pgs "github.com/lyft/protoc-gen-star"
	"github.com/lyft/protoc-gen-star/gogoproto"
)

var nonAlphaNumPattern = regexp.MustCompile("[^a-zA-Z0-9]")

func gogoType(f pgs.Field) (pgs.FilePath, TypeName, bool) {
	ft := f.Type()
	switch {
	case ft.IsMap():
		return "", "", false
	case ft.IsRepeated():
		return "", "", false
	case ft.IsEnum():
		return "", "", false
	case ft.IsEmbed():
		em := ft.Embed()
		if em.IsWellKnown() {
			switch em.WellKnownType() {
			case pgs.TimestampWKT:
				var stdtime bool
				ok, err := f.Extension(gogoproto.E_Stdtime, &stdtime)
				if err != nil {
					panic(fmt.Errorf("failed to parse stdtime extension value: %s", err))
				}
				if ok && stdtime {
					return pgs.FilePath("time"), TypeName("time.Time"), true
				}

			case pgs.DurationWKT:
				var stdduration bool
				ok, err := f.Extension(gogoproto.E_Stdduration, &stdduration)
				if err != nil {
					panic(fmt.Errorf("failed to parse stdduration extension value: %s", err))
				}
				if ok && stdduration {
					return pgs.FilePath("time"), TypeName("time.Duration"), true
				}
			}
		}
	}

	var customtype string
	hasCustomType, err := f.Extension(gogoproto.E_Customtype, &customtype)
	if err != nil {
		panic(fmt.Errorf("failed to parse customtype extension value: %s", err))
	}
	var casttype string
	hasCastType, err := f.Extension(gogoproto.E_Casttype, &casttype)
	if err != nil {
		panic(fmt.Errorf("failed to parse casttype extension value: %s", err))
	}
	if hasCustomType && hasCastType {
		panic(fmt.Errorf("both casttype and customtype specifed"))
	}
	if !hasCustomType && !hasCastType {
		return "", "", false
	}

	typeName := customtype
	if hasCastType {
		typeName = casttype
	}
	if i := strings.LastIndex(typeName, "."); i > 0 {
		pkg := typeName[:i]
		return pgs.FilePath(pkg), TypeName(fmt.Sprintf("%s.%s", nonAlphaNumPattern.ReplaceAllString(pkg, "_"), typeName[i+1:])), true
	}
	return "", TypeName(typeName), true
}

func (c context) PackageName(node pgs.Node) pgs.Name {
	e, ok := node.(pgs.Entity)
	if !ok {
		e = node.(pgs.Package).Files()[0]
	}

	_, pkg := c.optionPackage(e)

	// use import_path parameter ONLY if there is no go_package option in the file.
	if ip := c.p.Str("import_path"); ip != "" &&
		e.File().Descriptor().GetOptions().GetGoPackage() == "" {
		pkg = ip
	}

	// if the package name is a Go keyword, prefix with '_'
	if token.Lookup(pkg).IsKeyword() {
		pkg = "_" + pkg
	}

	// if package starts with digit, prefix with `_`
	if r, _ := utf8.DecodeRuneInString(pkg); unicode.IsDigit(r) {
		pkg = "_" + pkg
	}

	// package name is kosher
	return pgs.Name(pkg)
}

func (c context) FieldTypePackageName(f pgs.Field) pgs.Name {
	var en pgs.Entity
	switch ft := f.Type(); {
	case ft.IsEmbed():
		en = ft.Embed()
	case ft.IsEnum():
		en = ft.Enum()
	case ft.IsRepeated(), ft.IsMap():
		el := ft.Element()
		switch {
		case el.IsEmbed():
			en = el.Embed()
		case el.IsEnum():
			en = el.Enum()
		}
	default:
		return pgs.Name("")
	}
	return c.PackageName(en)
}

func (c gogoContext) FieldTypePackageName(f pgs.Field) pgs.Name {
	pkg, _, ok := gogoType(f)
	if !ok {
		return c.context.FieldTypePackageName(f)
	}
	return pgs.Name(nonAlphaNumPattern.ReplaceAllString(string(pkg), "_"))
}

func (c context) ImportPath(e pgs.Entity) pgs.FilePath {
	path, _ := c.optionPackage(e)
	path = c.p.Str("import_prefix") + path
	return pgs.FilePath(path)
}

func (c context) FieldTypeImportPath(f pgs.Field) pgs.FilePath {
	var en pgs.Entity
	switch ft := f.Type(); {
	case ft.IsEmbed():
		en = ft.Embed()
	case ft.IsEnum():
		en = ft.Enum()
	case ft.IsRepeated(), ft.IsMap():
		el := ft.Element()
		switch {
		case el.IsEmbed():
			en = el.Embed()
		case el.IsEnum():
			en = el.Enum()
		}
	default:
		return pgs.FilePath("")
	}
	return c.ImportPath(en)
}

func (c gogoContext) FieldTypeImportPath(f pgs.Field) pgs.FilePath {
	pkg, _, ok := gogoType(f)
	if !ok {
		return c.context.FieldTypeImportPath(f)
	}
	return pkg
}

func (c context) OutputPath(e pgs.Entity) pgs.FilePath {
	out := e.File().InputPath().SetExt(".pb.go")

	// source relative doesn't try to be fancy
	if Paths(c.p) == SourceRelative {
		return out
	}

	path, _ := c.optionPackage(e)

	// Import relative ignores the existing file structure
	return pgs.FilePath(path).Push(out.Base())
}

func (c context) optionPackage(e pgs.Entity) (path, pkg string) {
	// M mapping param overrides everything IFF the entity is not a build target
	if override, ok := c.p["M"+e.File().InputPath().String()]; ok && !e.BuildTarget() {
		path = override
		pkg = override
		if idx := strings.LastIndex(pkg, "/"); idx > -1 {
			pkg = pkg[idx+1:]
		}
		return
	}

	// check if there's a go_package option specified
	pkg = c.resolveGoPackageOption(e)
	path = e.File().InputPath().Dir().String()

	if pkg == "" {
		// have a proto package name, so use that
		if n := e.Package().ProtoName(); n != "" {
			pkg = n.SnakeCase().String()
		} else { // no other info, then replace all non-alphanumerics from the input file name
			pkg = nonAlphaNumPattern.ReplaceAllString(e.File().InputPath().BaseName(), "_")
		}
		return
	}

	// go_package="example.com/foo/bar;baz" should have a package name of `baz`
	if idx := strings.LastIndex(pkg, ";"); idx > -1 {
		path = pkg[:idx]
		pkg = nonAlphaNumPattern.ReplaceAllString(pkg[idx+1:], "_")
		return
	}

	// go_package="example.com/foo/bar" should have a package name of `bar`
	if idx := strings.LastIndex(pkg, "/"); idx > -1 {
		path = pkg
		pkg = nonAlphaNumPattern.ReplaceAllString(pkg[idx+1:], "_")
		return
	}

	pkg = nonAlphaNumPattern.ReplaceAllString(pkg, "_")

	return
}

func (c context) resolveGoPackageOption(e pgs.Entity) string {
	// attempt to get it from the current file
	if pkg := e.File().Descriptor().GetOptions().GetGoPackage(); pkg != "" {
		return pkg
	}

	// protoc-gen-go will use the go_package option from _any_ file in the same
	// execution since it's assumed that all the files are in the same Go
	// package. PG* will only verify this against files in the same proto package
	for _, f := range e.Package().Files() {
		if pkg := f.Descriptor().GetOptions().GetGoPackage(); pkg != "" {
			return pkg
		}
	}

	return ""
}

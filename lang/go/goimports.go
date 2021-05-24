package pgsgo

import (
	pgs "github.com/lyft/protoc-gen-star"
	"golang.org/x/tools/imports"
	"strings"
)

type goImports struct {
	filename string
}

// GoImports returns a PostProcessor that run goimports on any files ending . ".go"
func GoImports() pgs.PostProcessor { return goImports{} }

func (g goImports) Match(a pgs.Artifact) bool {
	var n string

	switch a := a.(type) {
	case pgs.GeneratorFile:
		n = a.Name
	case pgs.GeneratorTemplateFile:
		n = a.Name
	case pgs.CustomFile:
		n = a.Name
	case pgs.CustomTemplateFile:
		n = a.Name
	default:
		return false
	}

	g.filename = n

	return strings.HasSuffix(n, ".go")
}

func (g goImports) Process(in []byte) ([]byte, error) {
	return imports.Process(g.filename, in, nil)
}

package pgsgo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	pgs "github.com/vaidehi-figma/protoc-gen-star"
)

func TestContext_Params(t *testing.T) {
	t.Parallel()

	p := pgs.Parameters{}
	p.SetStr("foo", "bar")
	ctx := InitContext(p)

	params := ctx.Params()
	assert.Equal(t, "bar", params.Str("foo"))
}

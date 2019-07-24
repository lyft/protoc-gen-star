package pgs

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/stretchr/testify/assert"
)

func TestGetDependents(t *testing.T) {
	t.Parallel()

	pkg := dummyPkg()
	f := &file{
		pkg: pkg,
		desc: &descriptor.FileDescriptorProto{
			Package: proto.String(pkg.ProtoName().String()),
			Syntax:  proto.String(string(Proto3)),
			Name:    proto.String("test_file.proto"),
		},
	}

	m := &msg{parent: f}
	m.fqn = fullyQualifiedName(f, m)
	m2 := dummyMsg()
	deps := GetDependents([]Message{m, m2}, m.FullyQualifiedName())

	assert.Len(t, deps, 1)
	assert.Contains(t, deps, m2)

	deps = GetDependents([]Message{m, m2}, m2.FullyQualifiedName())
	assert.Len(t, deps, 1)
	assert.Contains(t, deps, m)
}

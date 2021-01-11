package pgs

import (
	"testing"

	"errors"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/stretchr/testify/assert"
)

func TestService_Name(t *testing.T) {
	t.Parallel()

	s := &service{desc: &descriptor.ServiceDescriptorProto{Name: proto.String("foo")}}
	assert.Equal(t, "foo", s.Name().String())
}

func TestService_FullyQualifiedName(t *testing.T) {
	t.Parallel()

	s := &service{fqn: "foo"}
	assert.Equal(t, s.fqn, s.FullyQualifiedName())
}

func TestService_Syntax(t *testing.T) {
	t.Parallel()

	s := &service{}
	f := dummyFile()
	f.AddService(s)

	assert.Equal(t, f.Syntax(), s.Syntax())
}

func TestService_Package(t *testing.T) {
	t.Parallel()

	s := &service{}
	f := dummyFile()
	f.AddService(s)

	assert.NotNil(t, s.Package())
	assert.Equal(t, f.Package(), s.Package())
}

func TestService_File(t *testing.T) {
	t.Parallel()

	s := &service{}
	f := dummyFile()
	f.AddService(s)

	assert.NotNil(t, s.File())
	assert.Equal(t, f, s.File())
}

func TestService_BuildTarget(t *testing.T) {
	t.Parallel()

	s := &service{}
	f := dummyFile()
	f.AddService(s)

	assert.False(t, s.BuildTarget())
	f.buildTarget = true
	assert.True(t, s.BuildTarget())
}

func TestService_Descriptor(t *testing.T) {
	t.Parallel()

	s := &service{desc: &descriptor.ServiceDescriptorProto{}}
	assert.Equal(t, s.desc, s.Descriptor())
}

func TestService_Extension(t *testing.T) {
	// cannot be parallel

	s := &service{desc: &descriptor.ServiceDescriptorProto{}}
	assert.NotPanics(t, func() { s.Extension(nil, nil) })
}

func TestService_Imports(t *testing.T) {
	t.Parallel()

	s := &service{}
	assert.Empty(t, s.Imports())
	s.AddMethod(&mockMethod{i: []File{&file{}}})
	assert.Len(t, s.Imports(), 1)

	nf := &file{desc: &descriptor.FileDescriptorProto{
		Name: proto.String("foobar"),
	}}
	s.AddMethod(&mockMethod{i: []File{nf, nf}})
	assert.Len(t, s.Imports(), 2)
}

func TestService_Methods(t *testing.T) {
	t.Parallel()

	s := &service{}
	assert.Empty(t, s.Methods())
	s.AddMethod(&method{})
	assert.Len(t, s.Methods(), 1)
}

func TestService_Accept(t *testing.T) {
	t.Parallel()

	s := &service{}
	s.AddMethod(&method{})

	assert.NoError(t, s.Accept(nil))

	v := &mockVisitor{}
	assert.NoError(t, s.Accept(v))
	assert.Equal(t, 1, v.service)
	assert.Zero(t, v.method)

	v.Reset()
	v.err = errors.New("fizz")
	v.v = v
	assert.Error(t, s.Accept(v))
	assert.Equal(t, 1, v.service)
	assert.Zero(t, v.method)

	v.Reset()
	assert.NoError(t, s.Accept(v))
	assert.Equal(t, 1, v.service)
	assert.Equal(t, 1, v.method)

	v.Reset()
	s.AddMethod(&mockMethod{err: errors.New("buzz")})
	assert.Error(t, s.Accept(v))
	assert.Equal(t, 1, v.service)
	assert.Equal(t, 2, v.method)
}

func TestService_ChildAtPath(t *testing.T) {
	t.Parallel()

	s := &service{}
	assert.Equal(t, s, s.ChildAtPath(nil))
	assert.Nil(t, s.ChildAtPath([]int32{0}))
	assert.Nil(t, s.ChildAtPath([]int32{0, 0}))
}

type mockService struct {
	Service
	i   []File
	f   File
	err error
}

func (s *mockService) Imports() []File { return s.i }

func (s *mockService) SetFile(f File) { s.f = f }

func (s *mockService) Accept(v Visitor) error {
	_, err := v.VisitService(s)
	if s.err != nil {
		return s.err
	}
	return err
}

func dummyService() *service {
	f := dummyFile()

	s := &service{
		desc: &descriptor.ServiceDescriptorProto{
			Name: proto.String("service"),
		},
	}

	f.AddService(s)
	return s
}

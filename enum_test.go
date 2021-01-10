package pgs

import (
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/stretchr/testify/assert"
)

func TestEnum_Name(t *testing.T) {
	t.Parallel()

	e := &enum{desc: &descriptor.EnumDescriptorProto{Name: proto.String("foo")}}
	assert.Equal(t, "foo", e.Name().String())
}

func TestEnum_FullyQualifiedName(t *testing.T) {
	t.Parallel()

	e := &enum{fqn: "enum"}
	assert.Equal(t, e.fqn, e.FullyQualifiedName())
}

func TestEnum_Syntax(t *testing.T) {
	t.Parallel()

	e := &enum{}
	f := dummyFile()
	f.AddEnum(e)

	assert.Equal(t, f.Syntax(), e.Syntax())
}

func TestEnum_Package(t *testing.T) {
	t.Parallel()

	e := &enum{}
	f := dummyFile()
	f.AddEnum(e)

	assert.NotNil(t, e.Package())
	assert.Equal(t, f.Package(), e.Package())
}

func TestEnum_File(t *testing.T) {
	t.Parallel()

	e := &enum{}
	m := dummyMsg()
	m.AddEnum(e)

	assert.NotNil(t, e.File())
	assert.Equal(t, m.File(), e.File())
}

func TestEnum_BuildTarget(t *testing.T) {
	t.Parallel()

	e := &enum{}
	f := dummyFile()
	f.AddEnum(e)

	assert.False(t, e.BuildTarget())
	f.buildTarget = true
	assert.True(t, e.BuildTarget())
}

func TestEnum_Descriptor(t *testing.T) {
	t.Parallel()

	e := &enum{desc: &descriptor.EnumDescriptorProto{}}
	assert.Equal(t, e.desc, e.Descriptor())
}

func TestEnum_Parent(t *testing.T) {
	t.Parallel()

	e := &enum{}
	f := dummyFile()
	f.AddEnum(e)

	assert.Equal(t, f, e.Parent())
}

func TestEnum_Imports(t *testing.T) {
	t.Parallel()
	assert.Nil(t, (&enum{}).Imports())
}

func TestEnum_Values(t *testing.T) {
	t.Parallel()

	e := &enum{}
	assert.Empty(t, e.Values())
	e.AddValue(&enumVal{})
	assert.Len(t, e.Values(), 1)
}

func TestEnum_Dependents(t *testing.T) {
	t.Parallel()

	t.Run("enum in file", func(t *testing.T) {
		t.Parallel()

		e := &enum{}
		f := dummyFile()
		f.AddEnum(e)

		assert.Empty(t, e.Dependents())
	})

	t.Run("enum in message", func(t *testing.T) {
		t.Parallel()

		e := &enum{}
		m := dummyMsg()
		m.AddEnum(e)

		assert.Empty(t, e.Dependents())
	})

	t.Run("external dependents", func(t *testing.T) {
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

		e := &enum{}
		e.fqn = fullyQualifiedName(f, e)
		m := dummyMsg()
		m.fqn = fullyQualifiedName(f, m)
		e.AddDependent(m)
		deps := e.Dependents()

		assert.Len(t, deps, 1)
		assert.Contains(t, deps, m)
	})
}

func TestEnum_Extension(t *testing.T) {
	// cannot be parallel

	e := &enum{desc: &descriptor.EnumDescriptorProto{}}
	assert.NotPanics(t, func() { e.Extension(nil, nil) })
}

func TestEnum_Accept(t *testing.T) {
	t.Parallel()

	e := &enum{}
	e.AddValue(&enumVal{})

	assert.NoError(t, e.Accept(nil))

	v := &mockVisitor{}
	assert.NoError(t, e.Accept(v))
	assert.Equal(t, 1, v.enum)
	assert.Zero(t, v.enumvalue)

	v.Reset()
	v.err = errors.New("")
	v.v = v
	assert.Error(t, e.Accept(v))
	assert.Equal(t, 1, v.enum)
	assert.Zero(t, v.enumvalue)

	v.Reset()
	assert.NoError(t, e.Accept(v))
	assert.Equal(t, 1, v.enum)
	assert.Equal(t, 1, v.enumvalue)

	v.Reset()
	e.AddValue(&mockEnumValue{err: errors.New("")})
	assert.Error(t, e.Accept(v))
	assert.Equal(t, 1, v.enum)
	assert.Equal(t, 2, v.enumvalue)
}

func TestEnum_ChildAtPath(t *testing.T) {
	t.Parallel()

	e := &enum{}
	assert.Equal(t, e, e.ChildAtPath(nil))
	assert.Nil(t, e.ChildAtPath([]int32{1}))
	assert.Nil(t, e.ChildAtPath([]int32{999, 123}))
}

type mockEnum struct {
	Enum
	p   ParentEntity
	err error
}

func (e *mockEnum) SetParent(p ParentEntity) { e.p = p }

func (e *mockEnum) Accept(v Visitor) error {
	_, err := v.VisitEnum(e)
	if e.err != nil {
		return e.err
	}
	return err
}

func dummyEnum() *enum {
	f := dummyFile()
	e := &enum{desc: &descriptor.EnumDescriptorProto{Name: proto.String("enum")}}
	f.AddEnum(e)
	return e
}

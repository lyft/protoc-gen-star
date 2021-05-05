package pgs

import (
	"errors"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/stretchr/testify/assert"
)

func TestField_Name(t *testing.T) {
	t.Parallel()

	f := &field{desc: &descriptor.FieldDescriptorProto{Name: proto.String("foo")}}

	assert.Equal(t, "foo", f.Name().String())
}

func TestField_FullyQualifiedName(t *testing.T) {
	t.Parallel()

	f := &field{fqn: "field"}
	assert.Equal(t, f.fqn, f.FullyQualifiedName())
}

func TestField_Syntax(t *testing.T) {
	t.Parallel()

	f := &field{}
	m := dummyMsg()
	m.AddField(f)

	assert.Equal(t, m.Syntax(), f.Syntax())
}

func TestField_Package(t *testing.T) {
	t.Parallel()

	f := &field{}
	m := dummyMsg()
	m.AddField(f)

	assert.NotNil(t, f.Package())
	assert.Equal(t, m.Package(), f.Package())
}

func TestField_File(t *testing.T) {
	t.Parallel()

	f := &field{}
	m := dummyMsg()
	m.AddField(f)

	assert.NotNil(t, f.File())
	assert.Equal(t, m.File(), f.File())
}

func TestField_BuildTarget(t *testing.T) {
	t.Parallel()

	f := &field{}
	m := dummyMsg()
	m.AddField(f)

	assert.False(t, f.BuildTarget())
	m.SetParent(&file{buildTarget: true})
	assert.True(t, f.BuildTarget())
}

func TestField_Descriptor(t *testing.T) {
	t.Parallel()

	f := &field{desc: &descriptor.FieldDescriptorProto{}}
	assert.Equal(t, f.desc, f.Descriptor())
}

func TestField_Message(t *testing.T) {
	t.Parallel()

	f := &field{}
	m := dummyMsg()
	m.AddField(f)

	assert.Equal(t, m, f.Message())
}

func TestField_OneOf(t *testing.T) {
	t.Parallel()

	f := &field{}
	assert.Nil(t, f.OneOf())
	assert.False(t, f.InOneOf())

	o := dummyOneof()
	o.AddField(f)

	assert.Equal(t, o, f.OneOf())
	assert.True(t, f.InOneOf())
}

func TestField_Type(t *testing.T) {
	t.Parallel()

	f := &field{}
	f.AddType(&scalarT{})

	assert.Equal(t, f.typ, f.Type())
}

func TestField_Extension(t *testing.T) {
	// cannot be parallel

	f := &field{desc: &descriptor.FieldDescriptorProto{}}
	assert.NotPanics(t, func() { f.Extension(nil, nil) })
}

func TestField_Accept(t *testing.T) {
	t.Parallel()

	f := &field{}

	assert.NoError(t, f.Accept(nil))

	v := &mockVisitor{err: errors.New("")}
	assert.Error(t, f.Accept(v))
	assert.Equal(t, 1, v.field)
}

func TestField_Imports(t *testing.T) {
	t.Parallel()

	f := &field{}
	f.AddType(&scalarT{})
	assert.Empty(t, f.Imports())

	f.AddType(&mockT{i: []File{&file{}, &file{}}})
	assert.Len(t, f.Imports(), 2)
}

func TestField_Required(t *testing.T) {
	t.Parallel()

	msg := dummyMsg()

	lbl := descriptor.FieldDescriptorProto_LABEL_REQUIRED

	f := &field{desc: &descriptor.FieldDescriptorProto{Label: &lbl}}
	f.SetMessage(msg)

	assert.False(t, f.Required(), "proto3 messages can never be marked required")

	f.File().(*file).desc.Syntax = proto.String(string(Proto2))
	assert.True(t, f.Required(), "proto2 + required")

	lbl = descriptor.FieldDescriptorProto_LABEL_OPTIONAL
	f.desc.Label = &lbl
	assert.False(t, f.Required(), "proto2 + optional")
}

func TestField_ChildAtPath(t *testing.T) {
	t.Parallel()

	f := &field{}
	assert.Equal(t, f, f.ChildAtPath(nil))
	assert.Nil(t, f.ChildAtPath([]int32{1}))
}

type mockField struct {
	Field
	i   []File
	m   Message
	err error
}

func (f *mockField) Imports() []File { return f.i }

func (f *mockField) SetMessage(m Message) { f.m = m }

func (f *mockField) Accept(v Visitor) error {
	_, err := v.VisitField(f)
	if f.err != nil {
		return f.err
	}
	return err
}

func dummyField() *field {
	m := dummyMsg()
	str := descriptor.FieldDescriptorProto_TYPE_STRING
	f := &field{desc: &descriptor.FieldDescriptorProto{Name: proto.String("field"), Type: &str}}
	m.AddField(f)
	t := &scalarT{}
	f.AddType(t)
	return f
}

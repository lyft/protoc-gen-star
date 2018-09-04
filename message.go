package pgs

import (
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
)

// Message describes a proto message, akin to a struct in Go. Messages can be
// contained in either another Message or File, and may house further Messages
// and/or Enums. While all Fields technically live on the Message, some may be
// contained within OneOf blocks.
type Message interface {
	ParentEntity

	// Descriptor returns the underlying proto descriptor for this message
	Descriptor() *generator.Descriptor

	// Parent returns either the File or Message that directly contains this
	// Message.
	Parent() ParentEntity

	// Fields returns all fields on the message, including those contained within
	// OneOf blocks.
	Fields() []Field

	// NonOneOfFields returns all fields not contained within OneOf blocks.
	NonOneOfFields() []Field

	// OneOfFields returns only the fields contained within OneOf blocks.
	OneOfFields() []Field

	// OneOfs returns the OneOfs contained within this Message.
	OneOfs() []OneOf

	// IsMapEntry identifies this message as a MapEntry. If true, this message is
	// not generated as code, and is used exclusively when marshaling a map field
	// to the wire format.
	IsMapEntry() bool

	// IsWellKnown identifies whether or not this Message is a WKT from the
	// `google.protobuf` package. Most official plugins special case these types
	// and they usually need to be handled differently.
	IsWellKnown() bool

	// WellKnownType returns the WellKnownType associated with this field. If
	// IsWellKnown returns false, UnknownWKT is returned.
	WellKnownType() WellKnownType

	setParent(p ParentEntity)
	addField(f Field)
	addOneOf(o OneOf)
}

type msg struct {
	parent ParentEntity

	msgs, preservedMsgs []Message
	enums               []Enum
	fields              []Field
	oneofs              []OneOf
	maps                []Message

	info SourceCodeInfo
}

func (m *msg) Name() Name                              { return Name(m.desc.GetName()) }
func (m *msg) FullyQualifiedName() string              { return fullyQualifiedName(m.parent, m) }
func (m *msg) Syntax() Syntax                          { return m.parent.Syntax() }
func (m *msg) Package() Package                        { return m.parent.Package() }
func (m *msg) File() File                              { return m.parent.File() }
func (m *msg) BuildTarget() bool                       { return m.parent.BuildTarget() }
func (m *msg) SourceCodeInfo() SourceCodeInfo          { return m.info }
func (m *msg) Descriptor() *descriptor.DescriptorProto { return m.desc }
func (m *msg) Parent() ParentEntity                    { return m.parent }
func (m *msg) IsMapEntry() bool                        { return m.desc.GetOptions().GetMapEntry() }
func (m *msg) Enums() []Enum                           { return m.enums }
func (m *msg) Messages() []Message                     { return m.msgs }
func (m *msg) Fields() []Field                         { return m.fields }
func (m *msg) OneOfs() []OneOf                         { return m.oneofs }
func (m *msg) MapEntries() []Message                   { return m.maps }

func (m *msg) WellKnownType() WellKnownType {
	if m.Package().ProtoName() == WellKnownTypePackage {
		return LookupWKT(m.Name())
	}
	return UnknownWKT
}

func (m *msg) IsWellKnown() bool {
	return m.WellKnownType().Valid()
}

func (m *msg) AllEnums() []Enum {
	es := m.Enums()
	for _, m := range m.msgs {
		es = append(es, m.AllEnums()...)
	}
	return es
}

func (m *msg) Messages() []Message {
	msgs := make([]Message, len(m.msgs))
	copy(msgs, m.msgs)
	return msgs
}

func (m *msg) AllMessages() []Message {
	msgs := m.Messages()
	for _, sm := range m.msgs {
		msgs = append(msgs, sm.AllMessages()...)
	}
	return msgs
}

func (m *msg) MapEntries() []Message {
	me := make([]Message, len(m.mapEntries))
	copy(me, m.mapEntries)
	return me
}

func (m *msg) Fields() []Field {
	f := make([]Field, len(m.fields))
	copy(f, m.fields)
	return f
}

func (m *msg) NonOneOfFields() (f []Field) {
	for _, fld := range m.fields {
		if !fld.InOneOf() {
			f = append(f, fld)
		}
	}
	return f
}

func (m *msg) OneOfFields() (f []Field) {
	for _, o := range m.oneofs {
		f = append(f, o.Fields()...)
	}

	return f
}

func (m *msg) OneOfs() []OneOf {
	o := make([]OneOf, len(m.oneofs))
	copy(o, m.oneofs)
	return o
}

func (m *msg) Imports() (i []File) {
	for _, f := range m.fields {
		i = append(i, f.Imports()...)
	}
	return
}

func (m *msg) Extension(desc *proto.ExtensionDesc, ext interface{}) (bool, error) {
	return extension(m.rawDesc.GetOptions(), desc, &ext)
}

func (m *msg) accept(v Visitor) (err error) {
	if v == nil {
		return nil
	}

	if v, err = v.VisitMessage(m); err != nil || v == nil {
		return
	}

	for _, e := range m.enums {
		if err = e.accept(v); err != nil {
			return
		}
	}

	for _, sm := range m.msgs {
		if err = sm.accept(v); err != nil {
			return
		}
	}

	for _, f := range m.fields {
		if err = f.accept(v); err != nil {
			return
		}
	}

	for _, o := range m.oneofs {
		if err = o.accept(v); err != nil {
			return
		}
	}

	return
}

func (m *msg) setParent(p ParentEntity) { m.parent = p }

func (m *msg) addEnum(e Enum) {
	e.setParent(m)
	m.enums = append(m.enums, e)
}

func (m *msg) addMessage(sm Message) {
	sm.setParent(m)
	m.msgs = append(m.msgs, sm)
}

func (m *msg) addField(f Field) {
	f.setMessage(m)
	m.fields = append(m.fields, f)
}

func (m *msg) addOneOf(o OneOf) {
	o.setMessage(m)
	m.oneofs = append(m.oneofs, o)
}

func (m *msg) addMapEntry(me Message) {
	me.setParent(m)
	m.mapEntries = append(m.mapEntries, me)

func (m *msg) childAtPath(path []int32) Entity {
	switch {
	case len(path) == 0:
		return m
	case len(path)%2 != 0:
		return nil
	}

	var child Entity
	switch path[0] {
	case messageTypeFieldPath:
		child = m.fields[path[1]]
	case messageTypeNestedTypePath:
		child = m.preservedMsgs[path[1]]
	case messageTypeEnumTypePath:
		child = m.enums[path[1]]
	case messageTypeOneofDeclPath:
		child = m.oneofs[path[1]]
	default:
		return nil
	}

	return child.childAtPath(path[2:])
}

func (m *msg) addSourceCodeInfo(info SourceCodeInfo) { m.info = info }

var _ Message = (*msg)(nil)

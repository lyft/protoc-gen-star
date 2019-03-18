package pgs

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/proto"
)

// An Extension is a custom option annotation that can be applied to an Entity to provide additional
// semantic details and metadata about the Entity. https://godoc.org/github.com/golang/protobuf/proto#ExtensionDesc
type Extension interface {
	Field

	// ParentEntity returns the ParentEntity where the Extension is defined
	ParentEntity() ParentEntity

	// Extendee returns the Message that the Extension is extending
	Extendee() Message

	// Number returns the tag number of the Extension
	Number() int32
}

type ext struct {
	desc     *descriptor.FieldDescriptorProto
	parent   ParentEntity
	extendee Message
	typ      FieldType

	info SourceCodeInfo
}

func (e *ext) Name() Name                                   { return Name(e.desc.GetName()) }
func (e *ext) FullyQualifiedName() string                   { return fullyQualifiedName(e.parent, e) }
func (e *ext) Syntax() Syntax                               { return e.parent.Syntax() }
func (e *ext) Package() Package                             { return e.parent.Package() }
func (e *ext) Imports() []File                              { return e.typ.Imports() }
func (e *ext) File() File                                   { return e.parent.File() }
func (e *ext) Extensions() []Extension                      { return nil }
func (e *ext) BuildTarget() bool                            { return e.parent.BuildTarget() }
func (e *ext) Descriptor() *descriptor.FieldDescriptorProto { return e.desc }
func (e *ext) SourceCodeInfo() SourceCodeInfo               { return e.info }
func (e *ext) Type() FieldType                              { return e.typ }
func (e *ext) ParentEntity() ParentEntity                   { return e.parent }
func (e *ext) Extendee() Message                            { return e.extendee }
func (e *ext) Number() int32                                { return e.desc.GetNumber() }
func (e *ext) Message() Message                             { return nil }
func (e *ext) InOneOf() bool                                { return false }
func (e *ext) OneOf() OneOf                                 { return nil }
func (e *ext) setMessage(m Message)                         {} // noop
func (e *ext) setOneOf(o OneOf)                             {} // noop
func (e *ext) addExtension(ext Extension)                   {} // noop

func (e *ext) Extension(desc *proto.ExtensionDesc, ext interface{}) (ok bool, err error) {
	return false, status.Error(codes.FailedPrecondition, "Extension cannot contain extensions.")
}

func (e *ext) Required() bool {
	return e.Syntax().SupportsRequiredPrefix() &&
		e.desc.GetLabel() == descriptor.FieldDescriptorProto_LABEL_REQUIRED
}

func (e *ext) accept(v Visitor) (err error) {
	if v == nil {
		return
	}

	_, err = v.VisitExtension(e)
	return
}

func (e *ext) addType(t FieldType) {
	t.setField(e)
	e.typ = t
}

func (e *ext) addSourceCodeInfo(info SourceCodeInfo) { e.info = info }

func (e *ext) childAtPath(path []int32) Entity {
	if len(path) == 0 {
		return e
	}
	return nil
}

var extractor extExtractor

func init() { extractor = protoExtExtractor{} }

type extExtractor interface {
	HasExtension(proto.Message, *proto.ExtensionDesc) bool
	GetExtension(proto.Message, *proto.ExtensionDesc) (interface{}, error)
}

type protoExtExtractor struct{}

func (e protoExtExtractor) HasExtension(pb proto.Message, ext *proto.ExtensionDesc) bool {
	return proto.HasExtension(pb, ext)
}

func (e protoExtExtractor) GetExtension(pb proto.Message, ext *proto.ExtensionDesc) (interface{}, error) {
	return proto.GetExtension(pb, ext)
}

func extension(opts proto.Message, e *proto.ExtensionDesc, out interface{}) (bool, error) {
	if opts == nil || reflect.ValueOf(opts).IsNil() {
		return false, nil
	}

	if e == nil {
		return false, errors.New("nil *proto.ExtensionDesc parameter provided")
	}

	if out == nil {
		return false, errors.New("nil extension output parameter provided")
	}

	o := reflect.ValueOf(out)
	if o.Kind() != reflect.Ptr {
		return false, errors.New("out parameter must be a pointer type")
	}

	if !extractor.HasExtension(opts, e) {
		return false, nil
	}

	val, err := extractor.GetExtension(opts, e)
	if err != nil || val == nil {
		return false, err
	}

	v := reflect.ValueOf(val)
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}

	for o.Kind() == reflect.Ptr || o.Kind() == reflect.Interface {
		if o.Kind() == reflect.Ptr && o.IsNil() {
			o.Set(reflect.New(o.Type().Elem()))
		}
		o = o.Elem()
	}

	if v.Type().AssignableTo(o.Type()) {
		o.Set(v)
		return true, nil
	}

	return true, fmt.Errorf("cannot assign extension type %q to output type %q",
		v.Type().String(),
		o.Type().String())
}

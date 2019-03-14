package pgs

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"reflect"

	"github.com/golang/protobuf/proto"
)

// An Extension is a custom option annotation that can be applied to an Entity to provide additional
// semantic details and metadata about the Entity. https://godoc.org/github.com/golang/protobuf/proto#ExtensionDesc
type Extension interface {

	// Descriptor returns the proto descriptor for this extension
	Descriptor() *descriptor.FieldDescriptorProto

	// File returns the File containing this extension
	File() File

	// potentially also include name, number, and type
	// https://github.com/golang/protobuf/blob/master/protoc-gen-go/descriptor/descriptor.proto#L177-L178
}

type ext struct {
	desc *descriptor.FieldDescriptorProto
	file File
}

func (e *ext) Descriptor() *descriptor.FieldDescriptorProto {
	return e.desc
}

func (e *ext) File() File {
	return e.file
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

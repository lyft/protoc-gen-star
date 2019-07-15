package pgs

import (
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// Enum describes an enumeration type. Its parent can be either a Message or a
// File.
type Enum interface {
	Entity

	// Descriptor returns the proto descriptor for this Enum
	Descriptor() *descriptor.EnumDescriptorProto

	// Parent resolves to either a Message or File that directly contains this
	// Enum.
	Parent() ParentEntity

	// Values returns each defined enumeration value.
	Values() []EnumValue

	addValue(v EnumValue)
	setParent(p ParentEntity)
	addDependent(ent Entity)
}

type enum struct {
	desc     *descriptor.EnumDescriptorProto
	parent   ParentEntity
	vals     []EnumValue
	info     SourceCodeInfo
	fqn      string
	deps     []Entity
	depCache []Entity
}

func (e *enum) Name() Name                                  { return Name(e.desc.GetName()) }
func (e *enum) FullyQualifiedName() string                  { return e.fqn }
func (e *enum) Syntax() Syntax                              { return e.parent.Syntax() }
func (e *enum) Package() Package                            { return e.parent.Package() }
func (e *enum) File() File                                  { return e.parent.File() }
func (e *enum) BuildTarget() bool                           { return e.parent.BuildTarget() }
func (e *enum) SourceCodeInfo() SourceCodeInfo              { return e.info }
func (e *enum) Descriptor() *descriptor.EnumDescriptorProto { return e.desc }
func (e *enum) Parent() ParentEntity                        { return e.parent }
func (e *enum) Imports() []File                             { return nil }
func (e *enum) Values() []EnumValue                         { return e.vals }

func (e *enum) Dependents() []Entity {
	if len(e.depCache) == 0 {
		set := make(map[string]Entity)

		set[e.resolveFQN(e.parent)] = e.parent
		for _, d := range e.parent.Dependents() {
			set[e.resolveFQN(d)] = d
		}
		for _, d := range e.deps {
			set[e.resolveFQN(d)] = d
			for _, dd := range d.Dependents() {
				set[e.resolveFQN(dd)] = dd
			}
		}

		e.depCache = make([]Entity, 0, len(set))
		for _, d := range set {
			e.depCache = append(e.depCache, d)
		}
	}
	return e.depCache
}

func (e *enum) resolveFQN(ent Entity) string {
	if f, ok := ent.(File); ok {
		return f.Name().String()
	}

	return ent.FullyQualifiedName()
}

func (e *enum) Extension(desc *proto.ExtensionDesc, ext interface{}) (bool, error) {
	return extension(e.desc.GetOptions(), desc, &ext)
}

func (e *enum) accept(v Visitor) (err error) {
	if v == nil {
		return nil
	}

	if v, err = v.VisitEnum(e); err != nil || v == nil {
		return
	}

	for _, ev := range e.vals {
		if err = ev.accept(v); err != nil {
			return
		}
	}

	return
}

func (e *enum) addValue(v EnumValue) {
	v.setEnum(e)
	e.vals = append(e.vals, v)
}

func (e *enum) setParent(p ParentEntity) { e.parent = p }

func (e *enum) addDependent(ent Entity) {
	e.deps = append(e.deps, ent)
}

func (e *enum) childAtPath(path []int32) Entity {
	switch {
	case len(path) == 0:
		return e
	case len(path)%2 != 0:
		return nil
	case path[0] == enumTypeValuePath:
		return e.vals[path[1]].childAtPath(path[2:])
	default:
		return nil
	}
}

func (e *enum) addSourceCodeInfo(info SourceCodeInfo) { e.info = info }

var _ Enum = (*enum)(nil)

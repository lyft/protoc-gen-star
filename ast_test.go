package pgs

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	descriptor "google.golang.org/protobuf/types/descriptorpb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	plugin_go "google.golang.org/protobuf/types/pluginpb"
)

func readCodeGenReq(t *testing.T, dir string) *plugin_go.CodeGeneratorRequest {
	filename := filepath.Join("testdata", "graph", dir, "code_generator_request.pb.bin")

	data, err := ioutil.ReadFile(filename)
	require.NoError(t, err, "unable to read CDR at %q", filename)

	req := &plugin_go.CodeGeneratorRequest{}
	err = proto.Unmarshal(data, req)
	require.NoError(t, err, "unable to unmarshal CDR data at %q", filename)

	return req
}

func readFileDescSet(t *testing.T, filename string) *descriptor.FileDescriptorSet {
	data, err := ioutil.ReadFile(filename)
	require.NoError(t, err, "unable to read FDS at %q", filename)

	fdset := &descriptor.FileDescriptorSet{}
	err = proto.Unmarshal(data, fdset)
	require.NoError(t, err, "unable to unmarshal FDS data at %q", filename)

	return fdset
}

func buildGraph(t *testing.T, dir string) AST {
	d := InitMockDebugger()
	ast := ProcessCodeGeneratorRequest(d, readCodeGenReq(t, dir))
	require.False(t, d.Failed(), "failed to build graph (see previous log statements)")
	return ast
}

func TestGraph_FDSet(t *testing.T) {
	fdset := readFileDescSet(t, "testdata/fdset.bin")
	d := InitMockDebugger()
	ast := ProcessFileDescriptorSet(d, fdset)

	require.False(t, d.Failed(), "failed to build graph from FDSet")
	msg, found := ast.Lookup(".kitchen.Sink")
	assert.True(t, found)
	assert.Implements(t, (*Message)(nil), msg)
}

func TestGraph_Messages(t *testing.T) {
	t.Parallel()
	g := buildGraph(t, "messages")

	tests := []struct {
		lookup                             string
		fldCt                              int
		isMap, isRepeated, isEmbed, isEnum bool
	}{
		{
			lookup: ".graph.messages.Scalars",
			fldCt:  15,
		},
		{
			lookup:  ".graph.messages.Embedded",
			fldCt:   6,
			isEmbed: true,
		},
		{
			lookup: ".graph.messages.Enums",
			fldCt:  6,
			isEnum: true,
		},
		{
			lookup:     ".graph.messages.Repeated",
			fldCt:      13,
			isRepeated: true,
		},
		{
			lookup: ".graph.messages.Maps",
			fldCt:  13,
			isMap:  true,
		},
		{
			lookup:  ".graph.messages.Recursive",
			fldCt:   1,
			isEmbed: true,
		},
	}

	for _, test := range tests {
		tc := test
		t.Run(tc.lookup, func(t *testing.T) {
			t.Parallel()

			ent, ok := g.Lookup(tc.lookup)
			require.True(t, ok, "unknown entity lookup")
			msg, ok := ent.(Message)
			require.True(t, ok, "entity is not a message")

			flds := msg.Fields()
			assert.Len(t, flds, tc.fldCt, "unexpected number of fields on the message")

			for _, fld := range flds {
				t.Run(fld.Name().String(), func(t *testing.T) {
					typ := fld.Type()
					assert.Equal(t, tc.isMap, typ.IsMap(), "should not be a map")
					assert.Equal(t, tc.isRepeated, typ.IsRepeated(), "should not be repeated")
					assert.Equal(t, tc.isEmbed, typ.IsEmbed(), "should not be embedded")
					assert.Equal(t, tc.isEnum, typ.IsEnum(), "should not be an enum")
				})
			}
		})
	}

	t.Run("oneof", func(t *testing.T) {
		t.Parallel()

		ent, ok := g.Lookup(".graph.messages.OneOfs")
		require.True(t, ok)
		msg, ok := ent.(Message)
		require.True(t, ok)

		flds := msg.Fields()
		oneofFlds := msg.OneOfFields()
		notOneofFlds := msg.NonOneOfFields()

		assert.Len(t, flds, 3)
		assert.Len(t, oneofFlds, 1)
		assert.Len(t, notOneofFlds, 2)

		oneofs := msg.OneOfs()
		require.Len(t, oneofs, 1)

		oo := oneofs[0]
		require.Len(t, oo.Fields(), 1)
		assert.Equal(t, int32(2), oo.Fields()[0].Descriptor().GetNumber())
		assert.Equal(t, oneofFlds, oo.Fields())
	})
}

func TestGraph_Services(t *testing.T) {
	t.Parallel()

	g := buildGraph(t, "services")

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		ent, ok := g.Lookup(".graph.services.Empty")
		require.True(t, ok)
		svc, ok := ent.(Service)
		require.True(t, ok)

		assert.Empty(t, svc.Methods())
	})

	t.Run("unary", func(t *testing.T) {
		t.Parallel()

		ent, ok := g.Lookup(".graph.services.Unary")
		require.True(t, ok)
		svc, ok := ent.(Service)
		require.True(t, ok)

		mtds := svc.Methods()
		assert.Len(t, mtds, 2)

		for _, mtd := range mtds {
			assert.False(t, mtd.ClientStreaming(), mtd.FullyQualifiedName())
			assert.False(t, mtd.ServerStreaming(), mtd.FullyQualifiedName())
		}
	})

	t.Run("streaming", func(t *testing.T) {
		t.Parallel()

		ent, ok := g.Lookup(".graph.services.Streaming")
		require.True(t, ok)
		svc, ok := ent.(Service)
		require.True(t, ok)

		mtds := svc.Methods()
		assert.Len(t, mtds, 3)

		tests := []struct{ client, server bool }{
			{true, false},
			{false, true},
			{true, true},
		}

		for i, mtd := range mtds {
			assert.Equal(t, tests[i].client, mtd.ClientStreaming(), mtd.FullyQualifiedName())
			assert.Equal(t, tests[i].server, mtd.ServerStreaming(), mtd.FullyQualifiedName())
		}
	})
}

func TestGraph_SourceCodeInfo(t *testing.T) {
	t.Parallel()

	g := buildGraph(t, "info")

	tests := map[string]string{
		"Info":                   "root message",
		"Info.Before":            "before message",
		"Info.BeforeEnum.BEFORE": "before enum value",
		"Info.field":             "field",
		"Info.Middle":            "middle message",
		"Info.Middle.inner":      "inner field",
		"Info.other_field":       "other field",
		"Info.After":             "after message",
		"Info.AfterEnum":         "after enum",
		"Info.AfterEnum.AFTER":   "after enum value",
		"Info.OneOf":             "oneof",
		"Info.oneof_field":       "oneof field",
		"Enum":                   "root enum comment",
		"Enum.ROOT":              "root enum value",
		"Service":                "service",
		"Service.Method":         "method",
	}

	for lookup, expected := range tests {
		t.Run(lookup, func(t *testing.T) {
			lo := ".graph.info." + lookup
			ent, ok := g.Lookup(lo)
			require.True(t, ok, "cannot find entity: %s", lo)
			info := ent.SourceCodeInfo()
			require.NotNil(t, info, "source code info is nil")
			assert.Contains(t, info.LeadingComments(), expected, "invalid leading comment")
		})
	}

	t.Run("file", func(t *testing.T) {
		f, ok := g.Targets()["info/info.proto"]
		require.True(t, ok, "cannot find file")

		info := f.SyntaxSourceCodeInfo()
		require.NotNil(t, info, "no source code info on syntax")
		assert.Contains(t, info.LeadingComments(), "syntax")
		assert.Equal(t, info, f.SourceCodeInfo(), "SourceCodeInfo should return SyntaxSourceCodeInfo")

		info = f.PackageSourceCodeInfo()
		require.NotNil(t, info, "no source code info on package")
		assert.Contains(t, info.LeadingComments(), "package")
	})
}

func TestGraph_MustSeen(t *testing.T) {
	t.Parallel()

	md := InitMockDebugger()
	g := &graph{
		d:        md,
		entities: make(map[string]Entity),
	}

	f := dummyFile()
	g.add(f)

	assert.Equal(t, f, g.mustSeen(g.resolveFQN(f)))
	assert.Nil(t, g.mustSeen(".foo.bar.baz"))
	assert.True(t, md.Failed())
}

func TestGraph_HydrateFieldType_Group(t *testing.T) {
	t.Parallel()

	md := InitMockDebugger()
	g := &graph{d: md}

	f := dummyField()
	f.Descriptor().Type = GroupT.ProtoPtr()

	assert.Nil(t, g.hydrateFieldType(f))
	assert.True(t, md.Failed())
}

func TestGraph_Packageless(t *testing.T) {
	t.Parallel()

	g := buildGraph(t, "packageless")

	tests := []struct {
		name        string
		entityIFace interface{}
	}{
		{".RootMessage", (*Message)(nil)},
		{".RootEnum", (*Enum)(nil)},
		{".RootMessage.field", (*Field)(nil)},
		{".RootEnum.VALUE", (*EnumValue)(nil)},
		{".RootMessage.NestedMsg", (*Message)(nil)},
		{".RootMessage.NestedEnum", (*Enum)(nil)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ent, ok := g.Lookup(tc.name)
			assert.True(t, ok)
			assert.NotNil(t, ent)
			assert.Implements(t, tc.entityIFace, ent)
		})
	}
}

func TestGraph_Extensions(t *testing.T) {
	t.Parallel()

	g := buildGraph(t, "extensions")
	assert.NotNil(t, g)

	ent, ok := g.Lookup("extensions/ext/data.proto")
	assert.True(t, ok)
	assert.NotNil(t, ent.(File).DefinedExtensions())
	assert.Len(t, ent.(File).DefinedExtensions(), 6)

	ent, ok = g.Lookup("extensions/everything.proto")
	assert.True(t, ok)
	assert.NotNil(t, ent.(File).DefinedExtensions())
	assert.Len(t, ent.(File).Imports(), 4)

	ent, ok = g.Lookup(".extensions.Request")
	assert.True(t, ok)
	assert.NotNil(t, ent.(Message).DefinedExtensions())
	assert.Len(t, ent.(Message).DefinedExtensions(), 1)

	ent, ok = g.Lookup(".google.protobuf.MessageOptions")
	assert.True(t, ok)
	assert.NotNil(t, ent.(Message).Extensions())
	assert.Len(t, ent.(Message).Extensions(), 1)
}

func TestGraph_Bidirectional(t *testing.T) {
	t.Parallel()

	fdset := readFileDescSet(t, "testdata/fdset.bin")
	d := InitMockDebugger()
	ast := ProcessFileDescriptorSetBidirectional(d, fdset)
	require.False(t, d.Failed(), "failed to build graph from FDSet")

	t.Run("nested", func(t *testing.T) {
		t.Parallel()

		finish, ok := ast.Lookup(".kitchen.Sink.Material.Finish")
		require.True(t, ok)

		kitchen, ok := ast.Lookup(".kitchen.Kitchen")
		require.True(t, ok)
		sink, ok := ast.Lookup(".kitchen.Sink")
		require.True(t, ok)

		material, ok := ast.Lookup(".kitchen.Sink.Material")
		require.True(t, ok)

		dependents := finish.(Enum).Dependents()
		require.Len(t, dependents, 3)
		assert.Contains(t, dependents, finish.(Enum).Parent())
		assert.Contains(t, dependents, sink)
		assert.Contains(t, dependents, kitchen)

		dependencies := sink.(Message).Dependencies()
		require.Len(t, dependencies, 2)
		assert.Contains(t, dependencies, material.(Message))
	})

	t.Run("files", func(t *testing.T) {
		t.Parallel()

		timestamp, ok := ast.Lookup("google/protobuf/timestamp.proto")
		require.True(t, ok)

		sinkProto, ok := ast.Lookup("kitchen/sink.proto")
		require.True(t, ok)
		kitchenProto, ok := ast.Lookup("kitchen/kitchen.proto")
		require.True(t, ok)

		dependents := timestamp.(File).Dependents()
		assert.Len(t, dependents, 2)
		assert.Contains(t, dependents, sinkProto)
		assert.Contains(t, dependents, kitchenProto)
	})
}

func TestGraph_Bidirectional_Messages_Enums(t *testing.T) {
	t.Parallel()

	d := InitMockDebugger()
	graph := ProcessCodeGeneratorRequestBidirectional(d, readCodeGenReq(t, "messages"))
	require.False(t, d.Failed(), "failed to build graph (see previous log statements)")

	t.Run("repeated", func(t *testing.T) {
		t.Parallel()

		beforeRepMsg, ok := graph.Lookup(".graph.messages.BeforeRepMsg")
		require.True(t, ok)
		repeated, ok := graph.Lookup(".graph.messages.Repeated")
		require.True(t, ok)

		dependents := beforeRepMsg.(Message).Dependents()
		require.Len(t, dependents, 1)
		assert.Contains(t, dependents, repeated)

		dependencies := repeated.(Message).Dependencies()
		require.Len(t, dependencies, 6)
		assert.Contains(t, dependents, repeated)

		beforeRepEnum, ok := graph.Lookup(".graph.messages.BeforeRepEnum")
		require.True(t, ok)

		dependents = beforeRepEnum.(Enum).Dependents()
		require.Len(t, dependents, 1)
		assert.Contains(t, dependents, repeated)
	})

	t.Run("message cycle", func(t *testing.T) {
		t.Parallel()

		recursiveMsg, ok := graph.Lookup(".graph.messages.Recursive")
		require.True(t, ok)
		assert.Empty(t, recursiveMsg.(Message).Dependents())
		assert.Empty(t, recursiveMsg.(Message).Dependencies())
	})

	t.Run("maps", func(t *testing.T) {
		t.Parallel()

		beforeMapMsg, ok := graph.Lookup(".graph.messages.BeforeMapMsg")
		require.True(t, ok)
		maps, ok := graph.Lookup(".graph.messages.Maps")
		require.True(t, ok)

		dependents := beforeMapMsg.(Message).Dependents()
		require.Len(t, dependents, 1)
		assert.Contains(t, dependents, maps)

		dependencies := maps.(Message).Dependencies()
		require.Len(t, dependencies, 6)
		assert.Contains(t, dependencies, beforeMapMsg.(Message))

		beforeMapEnum, ok := graph.Lookup(".graph.messages.BeforeMapEnum")
		require.True(t, ok)

		dependents = beforeMapEnum.(Enum).Dependents()
		require.Len(t, dependents, 1)
		assert.Contains(t, dependents, maps)
	})
}

func TestGraph_Bidirectional_Recursive(t *testing.T) {
	t.Parallel()

	d := InitMockDebugger()
	graph := ProcessCodeGeneratorRequestBidirectional(d, readCodeGenReq(t, "messages"))
	require.False(t, d.Failed(), "failed to build graph (see previous log statements)")

	tests := []struct {
		fqn      string
		expected []string
	}{
		{
			fqn:      ".graph.messages.Recursive",
			expected: []string{},
		},
		{
			fqn: ".graph.messages.Circular.Rock",
			expected: []string{
				".graph.messages.Circular.Paper",
				".graph.messages.Circular.Scissors",
			},
		},
		{
			fqn:      ".graph.messages.RepeatedRecursive",
			expected: []string{},
		},
	}

	for _, test := range tests {
		tc := test
		t.Run(tc.fqn, func(t *testing.T) {
			t.Parallel()

			m, ok := graph.Lookup(tc.fqn)
			require.True(t, ok)
			dependents := m.(Message).Dependents()
			dependencies := m.(Message).Dependencies()

			require.Len(t, dependents, len(tc.expected),
				"wanted %v, but got %v", tc.expected, dependents)
			require.Len(t, dependencies, len(tc.expected),
				"wanted %v, but got %v", tc.expected, dependents)

			set := make(map[string]bool)
			for _, name := range tc.expected {
				set[name] = true
			}

			for _, dep := range dependents {
				assert.Contains(t, set, dep.FullyQualifiedName())
			}
			for _, dep := range dependencies {
				assert.Contains(t, set, dep.FullyQualifiedName())
			}
		})
	}
}

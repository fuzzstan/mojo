package generator

import (
	"bytes"
	"fmt"
	"github.com/gogo/protobuf/proto"
	plugin "github.com/gogo/protobuf/protoc-gen-gogo/plugin"
	"github.com/mojo-lang/core/go/pkg/mojo"
	"github.com/mojo-lang/mojo/go/pkg/protobuf/descriptor"
	"github.com/mojo-lang/mojo/go/pkg/util"
	"io/ioutil"
	"log"
	"os"
	path2 "path"
	"sort"
	"strings"
)

// A Plugin provides functionality to add to the output during Go code generation,
// such as to produce RPC stubs.
type Plugin interface {
	// Name identifies the plugin.
	Name() string
	// Init is called once after data structures are built but before
	// code generation begins.
	Init(g *Generator)
	// Generate produces the code generated by the plugin for this file,
	// except for the imports, by calling the generator's methods P, In, and Out.
	Generate(file *descriptor.FileDescriptor)
	// GenerateImports produces the import declarations for this file.
	// It is called after Generate.
	GenerateImports(file *descriptor.FileDescriptor)
}

var plugins []Plugin

// Generator is the type whose methods generate the output, stored in the associated response structure.
type Generator struct {
	*bytes.Buffer

	Request  *plugin.CodeGeneratorRequest  // The input.
	Response *plugin.CodeGeneratorResponse // The output.

	Param        map[string]string // Command-line parameters.
	ImportPrefix string            // String to prefix to imported package file names.
	ImportMap    map[string]string // Mapping from .proto file name to import path

	Pkg map[string]string // The names under which we import support packages

	genFiles []*descriptor.FileDescriptor // Those files we will generate output for.

	allFiles       []*descriptor.FileDescriptor          // All files in the tree
	allFilesByName map[string]*descriptor.FileDescriptor // All files by filename.

	file *descriptor.FileDescriptor // The file we are compiling now.

	indent      string
	writeOutput bool
}

// New creates a new generator and allocates the request and response protobufs.
func New(files []*descriptor.FileDescriptor) *Generator {
	g := new(Generator)
	g.Buffer = new(bytes.Buffer)
	g.Request = new(plugin.CodeGeneratorRequest)
	g.Response = new(plugin.CodeGeneratorResponse)

	if len(files) > 0 {
		for _, file := range files {
			g.allFiles = append(g.allFiles, file)
		}
		g.genFiles = g.allFiles
	}

	return g
}

func (g *Generator) GetGeneratedFiles() []*descriptor.FileDescriptor {
	fileIndex := make(map[string]bool)
	for _, file := range g.Response.File {
		fileIndex[*file.Name] = true
	}

	var files []*descriptor.FileDescriptor
	for _, file := range g.genFiles {
		if fileIndex[*file.Name] {
			files = append(files, file)
		}
	}

	return files
}

// Error reports a problem, including an error, and exits the program.
func (g *Generator) Error(err error, msgs ...string) {
	s := strings.Join(msgs, " ") + ":" + err.Error()
	log.Print("protoc-gen-proto: error:", s)
	os.Exit(1)
}

// Fail reports a problem and exits the program.
func (g *Generator) Fail(msgs ...string) {
	s := strings.Join(msgs, " ")
	log.Print("protoc-gen-proto: error:", s)
	os.Exit(1)
}

// printAtom prints the (atomic, non-annotation) argument to the generated output.
func (g *Generator) printAtom(v interface{}) {
	switch v := v.(type) {
	case string:
		g.WriteString(v)
	case *string:
		g.WriteString(*v)
	case bool:
		fmt.Fprint(g, v)
	case *bool:
		fmt.Fprint(g, *v)
	case int:
		fmt.Fprint(g, v)
	case *int32:
		fmt.Fprint(g, *v)
	case *int64:
		fmt.Fprint(g, *v)
	case float64:
		fmt.Fprint(g, v)
	case *float64:
		fmt.Fprint(g, *v)
	default:
		g.Fail(fmt.Sprintf("unknown type in printer: %T", v))
	}
}

// P prints the arguments to the generated output.  It handles strings and int32s, plus
// handling indirections because they may be *string, etc.  Any inputs of type AnnotatedAtoms may emit
// annotations in a .meta file in addition to outputting the atoms themselves (if g.annotateCode
// is true).
func (g *Generator) P(strs ...interface{}) {
	if !g.writeOutput {
		return
	}
	g.WriteString(g.indent)
	g.S(strs...)
	g.WriteByte('\n')
}

func (g *Generator) S(strs ...interface{}) {
	if !g.writeOutput {
		return
	}
	for _, str := range strs {
		switch v := str.(type) {
		default:
			g.printAtom(v)
		}
	}
}

// deprecationComment is the standard comment added to deprecated
// messages, fields, enums, and enum values.
var deprecationComment = "// Deprecated: Do not use."

// PrintComments prints any comments from the source .proto file.
// The path is a comma-separated list of integers.
// It returns an indication of whether any comments were printed.
// See descriptor.proto for its format.
func (g *Generator) PrintComments(path string) bool {
	if !g.writeOutput {
		return false
	}
	if c, ok := g.makeComments(path); ok {
		g.P(c)
		return true
	}
	return false
}

// makeComments generates the comment string for the field, no "\n" at the end
func (g *Generator) makeComments(path string) (string, bool) {
	loc, ok := g.file.Comments[path]
	if !ok {
		return "", false
	}
	w := new(bytes.Buffer)
	nl := ""
	for _, line := range strings.Split(strings.TrimSuffix(loc.GetLeadingComments(), "\n"), "\n") {
		fmt.Fprintf(w, "%s//%s", nl, line)
		nl = "\n"
	}
	return w.String(), true
}

// In Indents the output one tab stop.
func (g *Generator) In() { g.indent += "    " }

// Out unindents the output one tab stop.
func (g *Generator) Out() {
	if len(g.indent) > 0 {
		g.indent = g.indent[4:]
	}
}

// WrapTypes walks the incoming data, wrapping DescriptorProtos, EnumDescriptorProtos
// and FileDescriptorProtos into file-referenced objects within the Generator.
// It also creates the list of files to generate and so should be called before GenerateAllFiles.
func (g *Generator) WrapTypes() {
	g.allFiles = make([]*descriptor.FileDescriptor, 0, len(g.Request.ProtoFile))
	g.allFilesByName = make(map[string]*descriptor.FileDescriptor, len(g.allFiles))
	genFileNames := make(map[string]bool)
	for _, n := range g.Request.FileToGenerate {
		genFileNames[n] = true
	}
	for _, f := range g.Request.ProtoFile {
		fd := &descriptor.FileDescriptor{
			FileDescriptorProto: f,
			Proto3:              descriptor.FileIsProto3(f),
		}

		// We must wrap the descriptors before we wrap the enums
		fd.Messages = descriptor.WrapMessageDescriptors(fd)
		g.buildNestedDescriptors(fd.Messages)

		fd.Enums = descriptor.WrapEnumDescriptors(fd, fd.Messages)
		g.buildNestedEnums(fd.Messages, fd.Enums)

		descriptor.ExtractComments(fd)
		g.allFiles = append(g.allFiles, fd)
		g.allFilesByName[f.GetName()] = fd
	}

	g.genFiles = make([]*descriptor.FileDescriptor, 0, len(g.Request.FileToGenerate))
	for _, fileName := range g.Request.FileToGenerate {
		fd := g.allFilesByName[fileName]
		if fd == nil {
			g.Fail("could not find file named", fileName)
		}
		g.genFiles = append(g.genFiles, fd)
	}
}

// Scan the descriptors in this file.  For each one, build the slice of nested descriptors
func (g *Generator) buildNestedDescriptors(descs []*descriptor.MessageDescriptor) {
	for _, desc := range descs {
		if len(desc.NestedType) != 0 {
			for _, nest := range descs {
				if nest.Parent == desc {
					desc.Messages = append(desc.Messages, nest)
				}
			}
			if len(desc.Messages) != len(desc.NestedType) {
				g.Fail("internal error: nesting failure for", desc.GetName())
			}
		}
	}
}

func (g *Generator) buildNestedEnums(descs []*descriptor.MessageDescriptor, enums []*descriptor.EnumDescriptor) {
	for _, desc := range descs {
		if len(desc.EnumType) != 0 {
			for _, enum := range enums {
				if enum.Parent == desc {
					desc.Enums = append(desc.Enums, enum)
				}
			}
			if len(desc.Enums) != len(desc.EnumType) {
				g.Fail("internal error: enum nesting failure for", desc.GetName())
			}
		}
	}
}

func (g *Generator) removeGeneratedDir(dir string) {
	var paths []string
	for _, file := range g.Response.File {
		path := path2.Dir(path2.Join(dir, *file.Name))
		paths = append(paths, path)
	}

	sort.Strings(paths)
	rootPath := ""
	for _, path := range paths {
		if len(rootPath) == 0 || !strings.HasPrefix(path, rootPath) {
			rootPath = path
			if util.IsExist(rootPath) {
				os.RemoveAll(rootPath)
			}
		}
	}
}

func (g *Generator) WriteAllFiles(dir string) {
	if g.Response == nil {
		return
	}

	g.removeGeneratedDir(dir)
	for _, file := range g.Response.File {
		if file.Name != nil && file.Content != nil {
			name := path2.Join(dir, *file.Name)
			path := path2.Dir(name)
			util.CreateDir(path)
			ioutil.WriteFile(name, []byte(*file.Content), 0666)
		}
	}
}

// GenerateAllFiles generates the output for all the files we're outputting.
func (g *Generator) GenerateAllFiles() *Generator {
	// Initialize the plugins
	for _, p := range plugins {
		p.Init(g)
	}
	// Generate the output. The generator runs for every file, even the files
	// that we don't generate output for, so that we can collate the full list
	// of exported symbols to support public imports.
	genFileMap := make(map[*descriptor.FileDescriptor]bool, len(g.genFiles))
	for _, file := range g.genFiles {
		genFileMap[file] = true
	}
	for _, file := range g.allFiles {
		g.Reset()
		g.writeOutput = genFileMap[file]
		if hasContent := g.generate(file); !hasContent {
			continue
		}
		if !g.writeOutput {
			continue
		}
		g.Response.File = append(g.Response.File, &plugin.CodeGeneratorResponse_File{
			Name:    proto.String(*file.Name),
			Content: proto.String(g.String()),
		})
	}

	return g
}

// Fill the response protocol buffer with the generated output for all the files we're
// supposed to generate.
func (g *Generator) generate(file *descriptor.FileDescriptor) bool {
	g.file = file

	isEmpty := true
	for _, enum := range g.file.Enums {
		if enum.Parent == nil {
			g.P()
			g.generateEnum(enum)
			isEmpty = false
		}
	}

	for _, message := range g.file.Messages {
		if message.Parent == nil {
			if message.Options != nil && message.Options.MapEntry != nil && *message.Options.MapEntry {
				continue
			}

			if isSystemMessage(message) && *file.Package == "mojo.core" {
				continue
			}

			g.P()
			g.generateMessage(message)
			isEmpty = false
		}
	}

	for _, service := range g.file.Services {
		g.P()
		g.generateService(service)
		isEmpty = false
	}

	// Run the plugins before the imports so we know which imports are necessary.
	//g.runPlugins(file)

	// Generate header and imports last, though they appear first in the output.
	rem := g.Buffer
	g.Buffer = new(bytes.Buffer)
	g.generateHeader()

	if len(g.file.Dependency) > 0 {
		g.P()
		g.generateImports()
	}

	if g.file.HasOptions() {
		options := g.file.Options
		if options.GoPackage != nil {
			g.P()
			g.WriteString("option go_package = \"")
			g.WriteString(*options.GoPackage)
			g.WriteString("\";")
		}

		if descriptor.HasJavaOptions(options) {
			if options.JavaMultipleFiles != nil {
				g.P()
				g.WriteString("option java_multiple_files = ")
				if *options.JavaMultipleFiles {
					g.WriteString("true;")
				} else {
					g.WriteString("false;")
				}
			}

			if options.JavaOuterClassname != nil {
				g.P()
				g.WriteString("option java_outer_classname = \"")
				g.WriteString(*options.JavaOuterClassname)
				g.WriteString("\";")
			}

			if options.JavaPackage != nil {
				g.P()
				g.WriteString("option java_package = \"")
				g.WriteString(*options.JavaPackage)
				g.WriteString("\";")
			}
		}
	}

	g.P()
	g.Write(rem.Bytes())

	return !isEmpty
}

// Generate the header, including package definition
func (g *Generator) generateHeader() {
	g.P("// Code generated by wand. DO NOT EDIT.")
	if g.file.GetOptions().GetDeprecated() {
		g.P("//")
		g.P("// ", g.file.Name, " is a deprecated file.")
	}

	g.P()

	syntax := "proto3"
	if g.file.Syntax != nil {
		syntax = *g.file.Syntax
	}
	g.P("syntax = \"", syntax, "\";")

	if g.file.Package != nil {
		g.P()
		g.P("package ", g.file.Package, ";")
	} else {
		// error
	}
}

func (g *Generator) generateImports() {
	for _, imp := range g.file.Dependency {
		g.P("import \"", imp, "\";")
	}
}

// Generate the enum definitions for this EnumDescriptor.
func (g *Generator) generateEnum(enum *descriptor.EnumDescriptor) {
	if enum.GetOptions().GetDeprecated() {
		g.P("// ", deprecationComment)
	}

	g.P("enum ", enum.Name, " {")
	g.In()
	for _, e := range enum.Value {
		if e.GetOptions().GetDeprecated() {
			g.P()
			g.P("// ", deprecationComment)
		}

		g.P(e.Name, "=", e.Number, ";")
	}
	g.Out()
	g.P("}")
}

// Generate the type, methods and default constant definitions for this Descriptor.
func (g *Generator) generateMessage(message *descriptor.MessageDescriptor) {
	g.P("message ", message.Name, " {")
	g.In()

	// Build a structure more suitable for generating the text in one pass
	for _, enum := range message.Enums {
		g.P()
		g.generateEnum(enum)
	}

	for _, msg := range message.Messages {
		if msg.Options != nil && msg.Options.MapEntry != nil && *msg.Options.MapEntry {
			continue
		}

		g.P()
		g.generateMessage(msg)
	}

	oneofs := make(map[int32]bool)

	printField := func(field *descriptor.FieldDescriptorProto) {
		g.WriteString(g.indent)
		if *field.Type == descriptor.FIELD_TYPE_MESSAGE {
			desc := message.GetInnerMessage(field.GetTypeName())
			if desc != nil && desc.GetOptions() != nil && desc.GetOptions().GetMapEntry() {
				keyType := descriptor.GetFieldTypeName(desc.Field[0])
				valType := descriptor.GetFieldTypeName(desc.Field[1])
				g.S("map<", keyType, ", ", valType, "> ", field.Name, " = ", field.Number)
			} else {
				if field.Label != nil && field.GetLabel() == descriptor.FIELD_LABEL_REPEATED {
					g.S("repeated ", descriptor.GetFieldTypeName(field), " ", field.GetName(), " = ", field.Number)
				} else {
					g.S(descriptor.GetFieldTypeName(field), " ", field.GetName(), " = ", field.Number)
				}
			}
		} else if field.Label != nil && field.GetLabel() == descriptor.FIELD_LABEL_REPEATED {
			g.S("repeated ", descriptor.GetFieldTypeName(field), " ", field.GetName(), " = ", field.Number)
		} else {
			g.S(descriptor.GetFieldTypeName(field), " ", field.GetName(), " = ", field.Number)
		}

		if field.Options != nil {
			g.S(" [")
			if alias := descriptor.GetStringFieldOption(field, mojo.E_Alias); len(alias) > 0 {
				g.S("(", mojo.E_Alias.Name, ")=", "\"", alias, "\"")
			}
			g.S("];\n")
		} else {
			g.S(";\n")
		}

		//fieldDeprecated := ""
		//if field.GetOptions().GetDeprecated() {
		//	fieldDeprecated = deprecationComment
		//}

		//fieldFullPath := fmt.Sprintf("%s,%d,%d", message.path, messageFieldPath, i)
		//c, ok := g.makeComments(fieldFullPath)
		//if ok {
		//	c += "\n"
		//}
	}

	for i, field := range message.Field {
		oneof := field.OneofIndex != nil
		if oneof && !oneofs[*field.OneofIndex] {
			odp := message.OneofDecl[int(*field.OneofIndex)]

			g.P("oneof ", odp.Name, " {")
			g.In()
			for j := i; j < len(message.Field); j++ {
				if message.Field[j].OneofIndex != nil {
					printField(message.Field[j])
				}
			}
			oneofs[*field.OneofIndex] = true

			g.Out()
			g.P("}")
		}

		if field.OneofIndex == nil {
			printField(field)
		}
	}

	g.Out()
	g.P("}")
}

// Generate the type, methods and default constant definitions for this Descriptor.
func (g *Generator) generateService(service *descriptor.ServiceDescriptor) {
	g.P("service ", service.Name, " {")
	g.In()

	for _, method := range service.Method {
		g.P("rpc ", method.Name, "(", method.InputType, ") returns (", method.OutputType, ");")
	}

	g.Out()
	g.P("}")
}

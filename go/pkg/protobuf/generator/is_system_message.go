package generator

import "github.com/mojo-lang/mojo/go/pkg/protobuf/descriptor"

var systemMessages = map[string]bool{
	"Bool":    true,
	"Int8":    true,
	"Int16":   true,
	"Int32":   true,
	"Int64":   true,
	"UInt8":   true,
	"UInt16":  true,
	"UInt32":  true,
	"UInt64":  true,
	"Int":     true,
	"UInt":    true,
	"Float32": true,
	"Float64": true,
	"String":  true,
	"Bytes":   true,
}

func isSystemMessage(msg *descriptor.MessageDescriptor) bool {
	return systemMessages[*msg.Name]
}

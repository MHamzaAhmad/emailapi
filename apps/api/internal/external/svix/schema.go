package svix

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ProtoToJSONSchema converts a proto.Message to JSON Schema Draft 7.
// This allows using proto definitions as the single source of truth for event schemas.
func ProtoToJSONSchema(msg proto.Message) map[string]interface{} {
	md := msg.ProtoReflect().Descriptor()
	return messageToSchema(md)
}

// messageToSchema converts a protoreflect.MessageDescriptor to JSON Schema.
func messageToSchema(md protoreflect.MessageDescriptor) map[string]interface{} {
	schema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}

	properties := schema["properties"].(map[string]interface{})
	fields := md.Fields()

	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		fieldName := string(fd.JSONName())
		properties[fieldName] = fieldToSchema(fd)
	}

	return schema
}

// fieldToSchema converts a protoreflect.FieldDescriptor to JSON Schema property.
func fieldToSchema(fd protoreflect.FieldDescriptor) map[string]interface{} {
	schema := map[string]interface{}{}

	// Add description from proto comments if available
	// Note: protoreflect doesn't expose comments directly, but we set meaningful type info

	// Handle repeated fields (arrays)
	if fd.IsList() {
		schema["type"] = "array"
		schema["items"] = kindToSchema(fd)
		return schema
	}

	// Handle map fields
	if fd.IsMap() {
		schema["type"] = "object"
		schema["additionalProperties"] = kindToSchema(fd.MapValue())
		return schema
	}

	// Handle single fields
	return kindToSchema(fd)
}

// kindToSchema converts a proto field kind to JSON Schema type.
func kindToSchema(fd protoreflect.FieldDescriptor) map[string]interface{} {
	schema := map[string]interface{}{}

	switch fd.Kind() {
	case protoreflect.BoolKind:
		schema["type"] = "boolean"

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		schema["type"] = "integer"
		schema["format"] = "int32"

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		schema["type"] = "integer"
		schema["format"] = "int64"

	case protoreflect.FloatKind:
		schema["type"] = "number"
		schema["format"] = "float"

	case protoreflect.DoubleKind:
		schema["type"] = "number"
		schema["format"] = "double"

	case protoreflect.StringKind:
		schema["type"] = "string"

	case protoreflect.BytesKind:
		schema["type"] = "string"
		schema["format"] = "byte"

	case protoreflect.EnumKind:
		schema["type"] = "string"
		// Could add enum values here from fd.Enum().Values()

	case protoreflect.MessageKind:
		// Handle well-known types
		fullName := fd.Message().FullName()
		switch fullName {
		case "google.protobuf.Timestamp":
			schema["type"] = "string"
			schema["format"] = "date-time"
		case "google.protobuf.Duration":
			schema["type"] = "string"
			schema["format"] = "duration"
		case "google.protobuf.Any":
			schema["type"] = "object"
		default:
			// Nested message - recurse
			return messageToSchema(fd.Message())
		}

	default:
		schema["type"] = "string"
	}

	return schema
}

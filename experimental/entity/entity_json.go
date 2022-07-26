package entity

import (
	"encoding/json"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

func init() { //nolint:gochecknoinits
	jsoniter.RegisterTypeEncoder("entity.EntityMessage", &entityMessageCodec{})
	//	jsoniter.RegisterTypeDecoder("entity.EntityMessage", &entityMessageCodec{})
}

type entityMessageCodec struct{}

// Custom writer that will send payload direclty as JSON when appropriate
func (u *EntityMessage) MarshalJSON() ([]byte, error) {
	cfg := jsoniter.ConfigCompatibleWithStandardLibrary
	stream := cfg.BorrowStream(nil)
	defer cfg.ReturnStream(stream)

	writeEntityMessageJSON(u, stream)
	if stream.Error != nil {
		return nil, stream.Error
	}

	buf := stream.Buffer()
	data := make([]byte, len(buf))
	copy(data, buf) // don't hold the internal jsoniter buffer
	return data, nil
}

func (codec *entityMessageCodec) IsEmpty(ptr unsafe.Pointer) bool {
	f := (*EntityMessage)(ptr)
	return f.Path == "" && f.Kind == "" && f.Payload == nil && f.Meta == nil
}

func (codec *entityMessageCodec) Encode(ptr unsafe.Pointer, stream *jsoniter.Stream) {
	writeEntityMessageJSON((*EntityMessage)(ptr), stream)
}

// func (codec *entityMessageCodec) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
// 	msg := EntityMessage{}
// 	err := fmt.Errorf("TODO")
// 	if err != nil {
// 		// keep existing iter error if it exists
// 		if iter.Error == nil {
// 			iter.Error = err
// 		}
// 		return
// 	}
// 	*((*EntityMessage)(ptr)) = msg
// }

func writeEntityMessageJSON(f *EntityMessage, stream *jsoniter.Stream) {
	stream.WriteObjectStart()
	stream.WriteObjectField("path")
	stream.WriteString(f.Path)

	if f.Kind != "" {
		stream.WriteMore()
		stream.WriteObjectField("kind")
		stream.WriteString(f.Kind)
	}

	if len(f.Payload) > 0 {
		stream.WriteMore()
		stream.WriteObjectField("payload")
		if json.Valid(f.Payload) {
			_, err := stream.Write(f.Payload)
			if err != nil {
				stream.Error = err
				return
			}
		} else {
			stream.WriteVal(f.Payload) // base64 encoded?
		}
	}

	if f.Meta != nil {
		stream.WriteMore()
		stream.WriteObjectField("meta")
		stream.WriteVal(f.Meta)
	}

	stream.WriteObjectEnd()
}

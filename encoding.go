package expose

import (
	"encoding/json"
	"io"
)

// Encoding is used for content negotiating. Request arguments and response values are encoded and decoded
// with the encoding that is matching the `Content-Type` or `Accept` header.
type Encoding struct {
	MimeType   string
	GetDecoder func(r io.Reader) Decoder
	GetEncoder func(w io.Writer) Encoder
}

type Decoder interface {
	Decode(v any) error
}

type DecoderFunc func(v any) error

func (f DecoderFunc) Decode(v any) error {
	return f(v)
}

type Encoder interface {
	Encode(v any) error
}
type EncoderFunc func(v any) error

func (f EncoderFunc) Encode(v any) error {
	return f(v)
}

var JsonEncoding = Encoding{
	MimeType: "application/json",
	GetEncoder: func(w io.Writer) Encoder {
		enc := json.NewEncoder(w)
		return EncoderFunc(func(v any) error {
			return enc.Encode(v)
		})
	},
	GetDecoder: func(r io.Reader) Decoder {
		dec := json.NewDecoder(r)

		return DecoderFunc(func(v any) error {
			return dec.Decode(v)
		})
	},
}

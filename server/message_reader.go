package gap

import (
	"encoding/binary"
	"io"
)

// MessageReader reads length-prefixed messages from a byte buffer.
// Each message is expected to be preceded by a 4-byte big-endian length prefix,
// matching the format used by StreamAdapter.Send and the client streaming transport.
type MessageReader struct {
	data   []byte
	offset int
}

// NewMessageReader creates a MessageReader over the given data.
func NewMessageReader(data []byte) *MessageReader {
	return &MessageReader{data: data}
}

// Next returns the next message from the buffer.
// Returns io.EOF when no more messages are available.
func (r *MessageReader) Next() ([]byte, error) {
	if r.offset >= len(r.data) {
		return nil, io.EOF
	}

	remaining := r.data[r.offset:]

	if len(remaining) < 4 {
		return nil, io.ErrUnexpectedEOF
	}

	length := binary.BigEndian.Uint32(remaining[:4])

	if len(remaining) < 4+int(length) {
		return nil, io.ErrUnexpectedEOF
	}

	msg := remaining[4 : 4+length]
	r.offset += 4 + int(length)

	return msg, nil
}

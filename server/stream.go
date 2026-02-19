package gap

import (
	"encoding/binary"
	"net/http"
)

// StreamAdapter provides length-prefixed streaming over HTTP responses.
// Each message is sent with a 4-byte big-endian length prefix followed by
// the protobuf-encoded message bytes.
type StreamAdapter struct {
	response http.ResponseWriter
}

func NewStreamAdapter(w http.ResponseWriter) *StreamAdapter {
	return &StreamAdapter{
		response: w,
	}
}

// SendHeaders writes streaming response headers and flushes them to the client.
func (sa *StreamAdapter) SendHeaders() error {
	sa.response.Header().Set("Content-Type", "application/x-protobuf-stream")
	sa.response.Header().Set("Transfer-Encoding", "chunked")
	sa.response.Header().Set("X-Content-Type-Options", "nosniff")
	sa.response.WriteHeader(http.StatusOK)

	if flusher, ok := sa.response.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

// Send writes a length-prefixed message to the stream.
func (sa *StreamAdapter) Send(data []byte) error {
	length := uint32(len(data))
	if err := binary.Write(sa.response, binary.BigEndian, length); err != nil {
		return err
	}

	_, err := sa.response.Write(data)
	if err != nil {
		return err
	}

	if flusher, ok := sa.response.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

package gapp

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
)

// UnaryHandler handles a unary RPC call. It receives the method name and request body,
// and returns the serialized response bytes or an error.
type UnaryHandler func(w http.ResponseWriter, r *http.Request, method string, body []byte) ([]byte, error)

// StreamHandler handles a streaming RPC call. It receives the method name, request body,
// and a StreamAdapter for sending responses. It should return nil after writing to the stream.
type StreamHandler func(w http.ResponseWriter, r *http.Request, method string, body []byte) error

// RpcHandler is the callback signature used by middleware and the dispatcher.
type RpcHandler func(w http.ResponseWriter, r *http.Request, method string, body []byte) ([]byte, error)

// Middleware wraps an RpcHandler, allowing pre/post processing of RPC calls.
type Middleware func(next RpcHandler) RpcHandler

// CORSConfig controls Cross-Origin Resource Sharing behavior.
type CORSConfig struct {
	AllowedOrigins []string                // specific origins, or ["*"] for all
	AllowOrigin    func(origin string) bool // dynamic check, takes precedence over AllowedOrigins
	AllowedHeaders []string                // defaults to standard RPC headers if nil
}

// DispatcherOption configures a Dispatcher.
type DispatcherOption func(*Dispatcher)

// WithCORS sets the CORS configuration for the dispatcher.
func WithCORS(config CORSConfig) DispatcherOption {
	return func(d *Dispatcher) {
		d.cors = &config
	}
}

// Dispatcher routes RPC calls to registered handlers.
type Dispatcher struct {
	Unary       map[string]UnaryHandler
	Streaming   map[string]StreamHandler
	middlewares []Middleware
	cors        *CORSConfig
}

// NewDispatcher creates a new Dispatcher with the given options.
func NewDispatcher(opts ...DispatcherOption) *Dispatcher {
	d := &Dispatcher{
		Unary:     make(map[string]UnaryHandler),
		Streaming: make(map[string]StreamHandler),
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Use adds a middleware to the dispatcher. Middlewares are applied in order:
// first added = outermost (runs first), last added = innermost (runs last, closest to handler).
func (d *Dispatcher) Use(m Middleware) {
	d.middlewares = append(d.middlewares, m)
}

// ServeHTTP implements http.Handler for the RPC dispatcher.
func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var handler RpcHandler = func(w http.ResponseWriter, r *http.Request, method string, body []byte) ([]byte, error) {
		if h, ok := d.Streaming[method]; ok {
			err := h(w, r, method, body)
			if err != nil {
				return nil, err
			}
			return nil, nil
		}
		if h, ok := d.Unary[method]; ok {
			return h(w, r, method, body)
		}
		return nil, ErrNotFound("unknown RPC method: " + method)
	}

	// Wrap with middleware: first added = outermost
	for i := len(d.middlewares) - 1; i >= 0; i-- {
		handler = d.middlewares[i](handler)
	}

	applyCORS(w, r, d.cors)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")

	method := r.Header.Get("X-Rpc-Method")

	body, bodyErr := io.ReadAll(r.Body)
	if bodyErr != nil {
		slog.Error("Failed to read request body", "error", bodyErr)
		writeRpcError(w, ErrValidation("Failed to read request body"))
		return
	}
	defer r.Body.Close()

	slog.Info("Handling RPC", "method", method)

	responseBytes, err := handler(w, r, method, body)

	if err != nil {
		slog.Error("Failed to handle request", "error", err, "method", method, "bodySize", len(body))

		var rpcErr *RpcError
		if errors.As(err, &rpcErr) {
			writeRpcError(w, rpcErr)
		} else {
			writeRpcError(w, ErrInternal("Internal server error"))
		}
		return
	}

	// If responseBytes is nil, the handler already wrote the response (e.g., streaming)
	if responseBytes == nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

func applyCORS(w http.ResponseWriter, r *http.Request, cors *CORSConfig) {
	origin := r.Header.Get("Origin")

	if cors == nil {
		// Default: reflect the request origin
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
	} else if origin != "" {
		allowed := false
		if cors.AllowOrigin != nil {
			allowed = cors.AllowOrigin(origin)
		} else {
			for _, o := range cors.AllowedOrigins {
				if o == "*" || o == origin {
					allowed = true
					break
				}
			}
		}
		if allowed {
			if len(cors.AllowedOrigins) == 1 && cors.AllowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
	}

	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	if cors != nil && len(cors.AllowedHeaders) > 0 {
		headers := ""
		for i, h := range cors.AllowedHeaders {
			if i > 0 {
				headers += ", "
			}
			headers += h
		}
		w.Header().Set("Access-Control-Allow-Headers", headers)
	} else {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With, X-Rpc-Method")
	}
}

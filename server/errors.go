package gap

import (
	"encoding/json"
	"net/http"
)

// Error codes for structured RPC error responses.
const (
	CodeValidationError = "VALIDATION_ERROR"
	CodeNotFound        = "NOT_FOUND"
	CodeAlreadyExists   = "ALREADY_EXISTS"
	CodeUnauthenticated = "UNAUTHENTICATED"
	CodePermissionDenied = "PERMISSION_DENIED"
	CodeRateLimited     = "RATE_LIMITED"
	CodeInternal        = "INTERNAL"
)

// RpcError is a structured error that serializes to JSON for RPC responses.
type RpcError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func (e *RpcError) Error() string {
	return e.Code + ": " + e.Message
}

// WithDetails returns a copy of the error with the given details added.
func (e *RpcError) WithDetails(details map[string]string) *RpcError {
	return &RpcError{
		Code:    e.Code,
		Message: e.Message,
		Details: details,
	}
}

func ErrValidation(msg string) *RpcError {
	return &RpcError{Code: CodeValidationError, Message: msg}
}

func ErrNotFound(msg string) *RpcError {
	return &RpcError{Code: CodeNotFound, Message: msg}
}

func ErrAlreadyExists(msg string) *RpcError {
	return &RpcError{Code: CodeAlreadyExists, Message: msg}
}

func ErrUnauthenticated(msg string) *RpcError {
	return &RpcError{Code: CodeUnauthenticated, Message: msg}
}

func ErrPermissionDenied(msg string) *RpcError {
	return &RpcError{Code: CodePermissionDenied, Message: msg}
}

func ErrRateLimited(msg string) *RpcError {
	return &RpcError{Code: CodeRateLimited, Message: msg}
}

func ErrInternal(msg string) *RpcError {
	return &RpcError{Code: CodeInternal, Message: msg}
}

func httpStatusForCode(code string) int {
	switch code {
	case CodeValidationError:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	case CodeAlreadyExists:
		return http.StatusConflict
	case CodeUnauthenticated:
		return http.StatusUnauthorized
	case CodePermissionDenied:
		return http.StatusForbidden
	case CodeRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func writeRpcError(w http.ResponseWriter, rpcErr *RpcError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusForCode(rpcErr.Code))
	json.NewEncoder(w).Encode(rpcErr)
}

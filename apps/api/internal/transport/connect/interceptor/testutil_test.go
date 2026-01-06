package interceptor

import (
	"context"
	"net/http"
	"reflect"
	"unsafe"

	"connectrpc.com/connect"
)

// newMockUnaryRequest creates a connect.AnyRequest with the given procedure and headers.
// Uses unsafe to set the unexported 'spec' field in connect.Request.
func newMockUnaryRequest(procedure string, headers map[string]string) connect.AnyRequest {
	req := connect.NewRequest(&struct{}{})

	// Set headers
	for k, v := range headers {
		req.Header().Set(k, v)
	}

	// Use reflect/unsafe to set the unexported 'spec' field
	// connect.Request struct structure: Msg, spec, header, peer
	val := reflect.ValueOf(req).Elem()
	field := val.FieldByName("spec")

	if field.IsValid() {
		reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).
			Elem().
			Set(reflect.ValueOf(connect.Spec{
				Procedure: procedure,
				IsClient:  false,
			}))
	}

	return req
}

// newMockStreamingConn creates a mock StreamingHandlerConn.
func newMockStreamingConn(procedure string, headers map[string]string) connect.StreamingHandlerConn {
	return &mockStreamingConn{
		procedure: procedure,
		headers:   headers,
	}
}

// mockStreamingConn implements connect.StreamingHandlerConn for testing.
type mockStreamingConn struct {
	connect.StreamingHandlerConn
	procedure string
	headers   map[string]string
}

func (m *mockStreamingConn) Spec() connect.Spec {
	return connect.Spec{
		Procedure: m.procedure,
	}
}

func (m *mockStreamingConn) RequestHeader() http.Header {
	h := make(http.Header)
	for k, v := range m.headers {
		h.Set(k, v)
	}
	return h
}

// newTestContext creates a context with user ID for testing.
func newTestContext(userID string) context.Context {
	return context.WithValue(context.Background(), ContextKeyUserID, userID)
}

package events

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// MockHandler implements Handler for testing.
type MockHandler struct {
	ctrl         *gomock.Controller
	handleCalled bool
	returnErr    error
	lastEvent    *SESEvent
}

func NewMockHandler(ctrl *gomock.Controller) *MockHandler {
	return &MockHandler{ctrl: ctrl}
}

func (m *MockHandler) Handle(ctx context.Context, event *SESEvent) error {
	m.handleCalled = true
	m.lastEvent = event
	return m.returnErr
}

// MockInboundHandler implements InboundHandler for testing.
type MockInboundHandler struct {
	ctrl         *gomock.Controller
	handleCalled bool
	returnErr    error
	lastBucket   string
	lastKey      string
}

func NewMockInboundHandler(ctrl *gomock.Controller) *MockInboundHandler {
	return &MockInboundHandler{ctrl: ctrl}
}

func (m *MockInboundHandler) HandleS3Event(ctx context.Context, bucket, key string) error {
	m.handleCalled = true
	m.lastBucket = bucket
	m.lastKey = key
	return m.returnErr
}

func TestRouter_Route_SendingEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("bounce event", func(t *testing.T) {
		router := NewRouter()
		handler := NewMockHandler(ctrl)
		router.RegisterHandler(EventTypeBounce, handler)

		event := SESEvent{
			EventType: "Bounce",
			Mail:      MailInfo{MessageId: "msg_1"},
			Bounce: &Bounce{
				BounceType: "Permanent",
				BouncedRecipients: []BouncedRecipient{
					{EmailAddress: "test@example.com"},
				},
			},
		}

		body, _ := json.Marshal(event)
		err := router.Route(ctx, string(body))
		require.NoError(t, err)
		assert.True(t, handler.handleCalled)
		assert.Equal(t, "Bounce", handler.lastEvent.EventType)
	})

	t.Run("complaint event", func(t *testing.T) {
		router := NewRouter()
		handler := NewMockHandler(ctrl)
		router.RegisterHandler(EventTypeComplaint, handler)

		event := SESEvent{
			EventType: "Complaint",
			Mail:      MailInfo{MessageId: "msg_1"},
			Complaint: &Complaint{
				ComplainedRecipients: []ComplainedRecipient{
					{EmailAddress: "spammer@example.com"},
				},
			},
		}

		body, _ := json.Marshal(event)
		err := router.Route(ctx, string(body))
		require.NoError(t, err)
		assert.True(t, handler.handleCalled)
	})

	t.Run("delivery event", func(t *testing.T) {
		router := NewRouter()
		handler := NewMockHandler(ctrl)
		router.RegisterHandler(EventTypeDelivery, handler)

		event := SESEvent{
			EventType: "Delivery",
			Mail:      MailInfo{MessageId: "msg_1"},
			Delivery: &Delivery{
				Recipients: []string{"recipient@example.com"},
			},
		}

		body, _ := json.Marshal(event)
		err := router.Route(ctx, string(body))
		require.NoError(t, err)
		assert.True(t, handler.handleCalled)
	})

	t.Run("unregistered event type", func(t *testing.T) {
		router := NewRouter()
		// No handler registered

		event := SESEvent{
			EventType: "Open",
			Mail:      MailInfo{MessageId: "msg_1"},
		}

		body, _ := json.Marshal(event)
		err := router.Route(ctx, string(body))
		require.NoError(t, err) // No error for unregistered types
	})

	t.Run("handler returns error", func(t *testing.T) {
		router := NewRouter()
		handler := NewMockHandler(ctrl)
		handler.returnErr = errors.New("handler failed")
		router.RegisterHandler(EventTypeBounce, handler)

		event := SESEvent{
			EventType: "Bounce",
			Mail:      MailInfo{MessageId: "msg_1"},
			Bounce:    &Bounce{BounceType: "Permanent"},
		}

		body, _ := json.Marshal(event)
		err := router.Route(ctx, string(body))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "handler failed")
	})
}

func TestRouter_Route_EventBridge_S3(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	router := NewRouter()
	inboundHandler := NewMockInboundHandler(ctrl)
	router.RegisterInboundHandler(inboundHandler, "inbound-emails-bucket")

	// EventBridge S3 event
	envelope := EventEnvelope{
		Version:    "0",
		ID:         "evt_1",
		DetailType: "Object Created",
		Source:     "aws.s3",
		Account:    "123456789",
		Region:     "us-east-1",
		Detail: map[string]interface{}{
			"bucket": map[string]interface{}{
				"name": "inbound-emails-bucket",
			},
			"object": map[string]interface{}{
				"key":  "emails/user_1/123",
				"size": 1024,
			},
		},
	}
	body, _ := json.Marshal(envelope)

	err := router.Route(ctx, string(body))
	require.NoError(t, err)
	assert.True(t, inboundHandler.handleCalled)
	assert.Equal(t, "inbound-emails-bucket", inboundHandler.lastBucket)
	assert.Equal(t, "emails/user_1/123", inboundHandler.lastKey)
}

func TestRouter_Route_EventBridge_S3_WrongBucket(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	router := NewRouter()
	inboundHandler := NewMockInboundHandler(ctrl)
	router.RegisterInboundHandler(inboundHandler, "inbound-emails-bucket")

	// S3 event from a different bucket
	envelope := EventEnvelope{
		Source: "aws.s3",
		Detail: map[string]interface{}{
			"bucket": map[string]interface{}{
				"name": "other-bucket",
			},
			"object": map[string]interface{}{
				"key": "some/key",
			},
		},
	}
	body, _ := json.Marshal(envelope)

	err := router.Route(ctx, string(body))
	require.NoError(t, err)
	assert.False(t, inboundHandler.handleCalled) // Handler should not be called
}

func TestRouter_Route_EventBridge_SES(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	router := NewRouter()
	handler := NewMockHandler(ctrl)
	router.RegisterHandler(EventTypeBounce, handler)

	// EventBridge SES event
	envelope := EventEnvelope{
		Source: "aws.ses",
		Detail: map[string]interface{}{
			"eventType": "Bounce",
			"mail": map[string]interface{}{
				"messageId": "msg_1",
			},
			"bounce": map[string]interface{}{
				"bounceType": "Permanent",
			},
		},
	}
	body, _ := json.Marshal(envelope)

	err := router.Route(ctx, string(body))
	require.NoError(t, err)
	assert.True(t, handler.handleCalled)
}

func TestRouter_Route_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	router := NewRouter()

	err := router.Route(ctx, "not valid json{")
	assert.Error(t, err)
}

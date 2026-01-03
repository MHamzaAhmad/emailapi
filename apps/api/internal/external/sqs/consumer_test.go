package sqs

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// mockClient implements Client for testing
type mockClient struct {
	messages     []Message
	receiveError error
	deleteError  error
	receiveCalls atomic.Int32
	deleteCalls  atomic.Int32
	messageIndex atomic.Int32
	returnEmpty  bool
}

func (m *mockClient) ReceiveMessages(ctx context.Context, maxMessages int, waitTimeSeconds int) ([]Message, error) {
	m.receiveCalls.Add(1)
	if m.receiveError != nil {
		return nil, m.receiveError
	}
	if m.returnEmpty {
		return nil, nil
	}
	// Return messages one at a time to control flow
	idx := int(m.messageIndex.Add(1)) - 1
	if idx >= len(m.messages) {
		m.returnEmpty = true
		return nil, nil
	}
	return []Message{m.messages[idx]}, nil
}

func (m *mockClient) DeleteMessage(ctx context.Context, receiptHandle string) error {
	m.deleteCalls.Add(1)
	return m.deleteError
}

func (m *mockClient) ChangeMessageVisibility(ctx context.Context, receiptHandle string, visibilityTimeoutSeconds int) error {
	return nil
}

// mockHandler implements MessageHandler for testing
type mockHandler struct {
	handleError   error
	handledBodies []string
}

func (m *mockHandler) Route(ctx context.Context, body string) error {
	m.handledBodies = append(m.handledBodies, body)
	return m.handleError
}

func TestConsumer_Start_GracefulShutdown(t *testing.T) {
	client := &mockClient{
		messages: []Message{
			{MessageID: "1", ReceiptHandle: "r1", Body: "body1"},
		},
	}
	handler := &mockHandler{}
	consumer := NewConsumer(client, handler, DefaultConsumerConfig())

	ctx, cancel := context.WithCancel(context.Background())

	// Start consumer in goroutine
	done := make(chan error)
	go func() {
		done <- consumer.Start(ctx)
	}()

	// Give it time to process
	time.Sleep(100 * time.Millisecond)

	// Cancel context
	cancel()

	// Should return nil on graceful shutdown
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil on graceful shutdown, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("consumer did not shutdown in time")
	}

	// Verify message was processed
	if len(handler.handledBodies) != 1 || handler.handledBodies[0] != "body1" {
		t.Errorf("expected handler to process body1, got: %v", handler.handledBodies)
	}

	// Verify message was deleted
	if client.deleteCalls.Load() != 1 {
		t.Errorf("expected 1 delete call, got: %d", client.deleteCalls.Load())
	}
}

func TestConsumer_Start_HandlerError_NoDelete(t *testing.T) {
	client := &mockClient{
		messages: []Message{
			{MessageID: "1", ReceiptHandle: "r1", Body: "body1"},
		},
	}
	handler := &mockHandler{
		handleError: errors.New("processing failed"),
	}
	consumer := NewConsumer(client, handler, DefaultConsumerConfig())

	ctx, cancel := context.WithCancel(context.Background())

	// Start consumer in goroutine
	done := make(chan error)
	go func() {
		done <- consumer.Start(ctx)
	}()

	// Give it time to process
	time.Sleep(100 * time.Millisecond)
	cancel()

	<-done

	// Message should NOT be deleted on handler error
	if client.deleteCalls.Load() != 0 {
		t.Errorf("expected 0 delete calls on handler error, got: %d", client.deleteCalls.Load())
	}
}

func TestConsumer_DefaultConfig(t *testing.T) {
	config := DefaultConsumerConfig()

	if config.MaxMessages != 10 {
		t.Errorf("expected MaxMessages=10, got: %d", config.MaxMessages)
	}
	if config.WaitTimeSeconds != 20 {
		t.Errorf("expected WaitTimeSeconds=20, got: %d", config.WaitTimeSeconds)
	}
	if config.InitialBackoff != 5*time.Second {
		t.Errorf("expected InitialBackoff=5s, got: %v", config.InitialBackoff)
	}
	if config.MaxBackoff != 60*time.Second {
		t.Errorf("expected MaxBackoff=60s, got: %v", config.MaxBackoff)
	}
}

func TestNewConsumer_AppliesDefaults(t *testing.T) {
	client := &mockClient{}
	handler := &mockHandler{}

	// Pass invalid config values
	consumer := NewConsumer(client, handler, ConsumerConfig{
		MaxMessages:     0,  // Invalid
		WaitTimeSeconds: -1, // Invalid
		InitialBackoff:  0,
		MaxBackoff:      0,
	})

	// Should have applied defaults
	if consumer.config.MaxMessages != 10 {
		t.Errorf("expected default MaxMessages=10, got: %d", consumer.config.MaxMessages)
	}
	if consumer.config.WaitTimeSeconds != 20 {
		t.Errorf("expected default WaitTimeSeconds=20, got: %d", consumer.config.WaitTimeSeconds)
	}
}

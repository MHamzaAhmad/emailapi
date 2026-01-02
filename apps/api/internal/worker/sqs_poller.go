package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/riverqueue/river"

	"github.com/emailapi/api/internal/events"
	"github.com/emailapi/api/internal/external/sqs"
)

// PollSQSArgs is the job arguments for the SQS poller.
// This is a periodic job that runs on a schedule.
type PollSQSArgs struct{}

func (PollSQSArgs) Kind() string { return "poll_sqs" }

// SQSPollerWorker polls SQS for email events and routes them to handlers.
type SQSPollerWorker struct {
	river.WorkerDefaults[PollSQSArgs]
	sqsClient sqs.Client
	router    *events.Router
}

// NewSQSPollerWorker creates a new SQS poller worker.
func NewSQSPollerWorker(sqsClient sqs.Client, router *events.Router) *SQSPollerWorker {
	return &SQSPollerWorker{
		sqsClient: sqsClient,
		router:    router,
	}
}

// Work processes SQS messages. Called periodically by River.
func (w *SQSPollerWorker) Work(ctx context.Context, job *river.Job[PollSQSArgs]) error {
	// Long-poll for messages (up to 20 seconds)
	messages, err := w.sqsClient.ReceiveMessages(ctx, 10, 20)
	if err != nil {
		return fmt.Errorf("failed to receive SQS messages: %w", err)
	}

	if len(messages) == 0 {
		return nil // No messages, nothing to do
	}

	fmt.Printf("Received %d SQS messages\n", len(messages))

	// Process each message
	for _, msg := range messages {
		start := time.Now()

		if err := w.router.Route(ctx, msg.Body); err != nil {
			// Log error but continue processing other messages
			// Message will become visible again after visibility timeout
			fmt.Printf("Failed to process message %s: %v\n", msg.MessageID, err)
			continue
		}

		// Successfully processed - delete from queue
		if err := w.sqsClient.DeleteMessage(ctx, msg.ReceiptHandle); err != nil {
			fmt.Printf("Failed to delete message %s: %v\n", msg.MessageID, err)
			// Don't return error - message was processed, just not deleted
			// It will be deleted on next successful receive
		}

		fmt.Printf("Processed message %s in %v\n", msg.MessageID, time.Since(start))
	}

	return nil
}

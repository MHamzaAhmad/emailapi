package sqs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// client implements the Client interface using AWS SDK v2.
type client struct {
	sqsClient *sqs.Client
	queueURL  string
}

// New creates a new SQS client.
func New(cfg aws.Config, queueURL string) Client {
	return &client{
		sqsClient: sqs.NewFromConfig(cfg),
		queueURL:  queueURL,
	}
}

// ReceiveMessages long-polls for messages from the SQS queue.
func (c *client) ReceiveMessages(ctx context.Context, maxMessages int, waitTimeSeconds int) ([]Message, error) {
	input := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.queueURL),
		MaxNumberOfMessages: int32(maxMessages),
		WaitTimeSeconds:     int32(waitTimeSeconds),
		// Request all message attributes
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameAll,
		},
		MessageSystemAttributeNames: []types.MessageSystemAttributeName{
			types.MessageSystemAttributeNameAll,
		},
	}

	output, err := c.sqsClient.ReceiveMessage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to receive messages: %w", err)
	}

	messages := make([]Message, len(output.Messages))
	for i, msg := range output.Messages {
		attrs := make(map[string]string)
		for k, v := range msg.Attributes {
			attrs[string(k)] = v
		}

		messages[i] = Message{
			MessageID:     aws.ToString(msg.MessageId),
			ReceiptHandle: aws.ToString(msg.ReceiptHandle),
			Body:          aws.ToString(msg.Body),
			Attributes:    attrs,
		}

		// Extract MessageGroupId for FIFO queues
		if groupID, ok := msg.Attributes["MessageGroupId"]; ok {
			messages[i].MessageGroupID = groupID
		}
	}

	return messages, nil
}

// DeleteMessage removes a message from the queue.
func (c *client) DeleteMessage(ctx context.Context, receiptHandle string) error {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	}

	_, err := c.sqsClient.DeleteMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// ChangeMessageVisibility extends the visibility timeout for a message.
func (c *client) ChangeMessageVisibility(ctx context.Context, receiptHandle string, visibilityTimeoutSeconds int) error {
	input := &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(c.queueURL),
		ReceiptHandle:     aws.String(receiptHandle),
		VisibilityTimeout: int32(visibilityTimeoutSeconds),
	}

	_, err := c.sqsClient.ChangeMessageVisibility(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to change message visibility: %w", err)
	}

	return nil
}

package gatewayclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/mohsin123321/notification-system/internal/domain/gateway/pure"
	notificationv1 "github.com/mohsin123321/notification-system/internal/gen/proto/notification/v1"
)

// Client is the infra adapter for the notification gRPC service. It implements
// domain/gateway/port.NotifierClient — injected directly into the gateway app
// with no adapter boilerplate in cmd/.
type Client struct {
	conn *grpc.ClientConn
	stub notificationv1.NotificationServiceClient
}

// New creates a Client connected to the given address.
func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, stub: notificationv1.NewNotificationServiceClient(conn)}, nil
}

// Close tears down the underlying gRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Send implements port.NotifierClient.
func (c *Client) Send(ctx context.Context, userID, notifType, idempotencyKey string, payload []byte) (*pure.SendResult, error) {
	resp, err := c.stub.SendNotification(ctx, &notificationv1.SendNotificationRequest{
		UserId:         userID,
		Type:           notifType,
		IdempotencyKey: idempotencyKey,
		Payload:        payload,
	})
	if err != nil {
		return nil, err
	}

	result := &pure.SendResult{NotificationID: resp.GetNotificationId()}
	for _, d := range resp.GetDecisions() {
		result.Decisions = append(result.Decisions, pure.ChannelDecision{
			Channel: d.GetChannel(),
			Status:  d.GetStatus(),
			Reason:  d.GetReason(),
		})
	}
	return result, nil
}

// Get implements port.NotifierClient.
func (c *Client) Get(ctx context.Context, id string) (*pure.NotificationDetail, error) {
	resp, err := c.stub.GetNotification(ctx, &notificationv1.GetNotificationRequest{
		NotificationId: id,
	})
	if err != nil {
		return nil, err
	}

	detail := &pure.NotificationDetail{
		NotificationID: resp.GetNotificationId(),
		UserID:         resp.GetUserId(),
		Type:           resp.GetType(),
		CreatedAt:      resp.GetCreatedAt().AsTime(),
	}
	for _, d := range resp.GetDeliveries() {
		detail.Deliveries = append(detail.Deliveries, pure.DeliveryDetail{
			Channel:   d.GetChannel(),
			Status:    d.GetStatus(),
			Attempts:  int(d.GetAttempts()),
			LastError: d.GetLastError(),
			UpdatedAt: d.GetUpdatedAt().AsTime(),
		})
	}
	return detail, nil
}

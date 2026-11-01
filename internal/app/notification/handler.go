package notification

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	notifservice "github.com/mohsin123321/notification-system/internal/app/notification/service"
	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
	notificationv1 "github.com/mohsin123321/notification-system/internal/gen/proto/notification/v1"
)

type handler struct {
	notificationv1.UnimplementedNotificationServiceServer
	svc *notifservice.Service
}

func newHandler(svc *notifservice.Service) *handler {
	return &handler{svc: svc}
}

func (h *handler) SendNotification(ctx context.Context, req *notificationv1.SendNotificationRequest) (*notificationv1.SendNotificationResponse, error) {
	var payload map[string]any
	if len(req.GetPayload()) > 0 {
		if err := json.Unmarshal(req.GetPayload(), &payload); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid payload JSON: %v", err)
		}
	}

	notif, err := h.svc.Send(ctx, notifservice.SendRequest{
		UserID:         pure.UserID(req.GetUserId()),
		Type:           pure.NotificationType(req.GetType()),
		IdempotencyKey: req.GetIdempotencyKey(),
		Payload:        payload,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "send notification: %v", err)
	}

	resp := &notificationv1.SendNotificationResponse{
		NotificationId: string(notif.ID),
	}
	for _, d := range notif.Deliveries {
		resp.Decisions = append(resp.Decisions, &notificationv1.ChannelDecision{
			Channel: string(d.Channel),
			Status:  string(d.Status),
		})
	}
	return resp, nil
}

func (h *handler) GetNotification(ctx context.Context, req *notificationv1.GetNotificationRequest) (*notificationv1.GetNotificationResponse, error) {
	notif, err := h.svc.Get(ctx, pure.NotificationID(req.GetNotificationId()))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "notification not found: %v", err)
	}

	resp := &notificationv1.GetNotificationResponse{
		NotificationId: string(notif.ID),
		UserId:         string(notif.UserID),
		Type:           string(notif.Type),
		CreatedAt:      timestamppb.New(notif.CreatedAt),
	}
	for _, d := range notif.Deliveries {
		resp.Deliveries = append(resp.Deliveries, &notificationv1.DeliveryStatus{
			Channel:   string(d.Channel),
			Status:    string(d.Status),
			Attempts:  int32(d.Attempts),
			LastError: d.LastError,
			UpdatedAt: timestamppb.New(d.UpdatedAt),
		})
	}
	return resp, nil
}

func (h *handler) WatchNotificationStatus(req *notificationv1.WatchNotificationStatusRequest, stream grpc.ServerStreamingServer[notificationv1.StatusEvent]) error {
	id := pure.NotificationID(req.GetNotificationId())
	ctx := stream.Context()

	ch, err := h.svc.WatchStatus(ctx, id)
	if err != nil {
		return status.Errorf(codes.Internal, "watch status: %v", err)
	}
	defer h.svc.UnwatchStatus(ctx, id, ch)

	for {
		select {
		case event, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(&notificationv1.StatusEvent{
				NotificationId: string(event.NotificationID),
				Channel:        string(event.Channel),
				Status:         string(event.Status),
				Error:          event.Error,
				UpdatedAt:      timestamppb.New(event.UpdatedAt),
			}); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

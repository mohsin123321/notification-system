package gateway

import (
	"context"
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mohsin123321/notification-system/internal/domain/gateway/pure"
)

// ServiceIface is what the handler needs from the service layer.
// Defined here — consumer owns the interface.
type ServiceIface interface {
	Send(ctx context.Context, userID, notifType, idempotencyKey string, payload []byte) (*pure.SendResult, error)
	Get(ctx context.Context, id string) (*pure.NotificationDetail, error)
}

type handler struct {
	svc ServiceIface
}

func newHandler(svc ServiceIface) *handler {
	return &handler{svc: svc}
}

func (h *handler) handleSend(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		http.Error(w, `{"error":"Idempotency-Key header required"}`, http.StatusBadRequest)
		return
	}

	var req SendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
		return
	}

	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		http.Error(w, `{"error":"failed to marshal payload"}`, http.StatusInternalServerError)
		return
	}
	result, err := h.svc.Send(r.Context(), req.UserID, req.Type, idempotencyKey, payloadBytes)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	resp := SendResponse{NotificationID: result.NotificationID}
	for _, d := range result.Decisions {
		resp.Decisions = append(resp.Decisions, DecisionResponse{
			Channel: d.Channel,
			Status:  d.Status,
			Reason:  d.Reason,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

func (h *handler) handleGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, `{"error":"missing notification id"}`, http.StatusBadRequest)
		return
	}

	detail, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	resp := GetResponse{
		NotificationID: detail.NotificationID,
		UserID:         detail.UserID,
		Type:           detail.Type,
		CreatedAt:      detail.CreatedAt,
	}
	for _, d := range detail.Deliveries {
		resp.Deliveries = append(resp.Deliveries, DeliveryResponse{
			Channel:   d.Channel,
			Status:    d.Status,
			Attempts:  d.Attempts,
			LastError: d.LastError,
			UpdatedAt: d.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(grpcCodeToHTTP(st.Code()))
	json.NewEncoder(w).Encode(map[string]string{"error": st.Message()}) //nolint:errcheck
}

func grpcCodeToHTTP(code codes.Code) int {
	switch code {
	case codes.NotFound:
		return http.StatusNotFound
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

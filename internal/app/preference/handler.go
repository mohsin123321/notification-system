package preference

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	prefservice "github.com/mohsin123321/notification-system/internal/app/preference/service"
	usersv1 "github.com/mohsin123321/notification-system/internal/gen/proto/users/v1"
)

type handler struct {
	usersv1.UnimplementedUsersServiceServer
	svc *prefservice.Service
}

func newHandler(svc *prefservice.Service) *handler {
	return &handler{svc: svc}
}

func (h *handler) GetPreferences(ctx context.Context, req *usersv1.GetPreferencesRequest) (*usersv1.GetPreferencesResponse, error) {
	prefs, err := h.svc.GetPreferences(ctx, req.GetUserId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get preferences: %v", err)
	}

	resp := &usersv1.GetPreferencesResponse{
		Email: prefs.Email,
		Phone: prefs.Phone,
	}

	for _, cp := range prefs.ChannelPrefs {
		resp.Channels = append(resp.Channels, &usersv1.ChannelPreference{
			Channel: cp.Channel,
			Enabled: cp.Enabled,
		})
	}

	if qh := prefs.QuietHours; qh.Enabled {
		resp.QuietHours = &usersv1.QuietHours{
			Enabled:   qh.Enabled,
			StartTime: qh.StartTime,
			EndTime:   qh.EndTime,
			Timezone:  qh.Timezone,
		}
	}

	return resp, nil
}

func (h *handler) RegisterDevice(ctx context.Context, req *usersv1.RegisterDeviceRequest) (*usersv1.RegisterDeviceResponse, error) {
	if err := h.svc.RegisterDevice(ctx, req.GetUserId(), req.GetDeviceId()); err != nil {
		return nil, status.Errorf(codes.Internal, "register device: %v", err)
	}
	return &usersv1.RegisterDeviceResponse{
		DeviceId: req.GetDeviceId(),
		UserId:   req.GetUserId(),
	}, nil
}

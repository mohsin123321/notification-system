package userclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	usersv1 "github.com/mohsin123321/notification-system/internal/gen/proto/users/v1"
	"github.com/mohsin123321/notification-system/internal/domain/notification/pure"
)

// Client wraps the users gRPC service and implements notification/port.PreferenceClient.
type Client struct {
	conn *grpc.ClientConn
	stub usersv1.UsersServiceClient
}

// New creates a new Client connected to the given address.
func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, stub: usersv1.NewUsersServiceClient(conn)}, nil
}

// Close tears down the underlying gRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// GetPreferences fetches user preferences from the users service and maps them
// to notification/pure.Preferences.
func (c *Client) GetPreferences(ctx context.Context, userID pure.UserID) (*pure.Preferences, error) {
	resp, err := c.stub.GetPreferences(ctx, &usersv1.GetPreferencesRequest{UserId: string(userID)})
	if err != nil {
		return nil, err
	}

	prefs := &pure.Preferences{
		Email: resp.GetEmail(),
		Phone: resp.GetPhone(),
	}

	for _, ch := range resp.GetChannels() {
		prefs.Channels = append(prefs.Channels, pure.ChannelPref{
			Channel: pure.Channel(ch.GetChannel()),
			Enabled: ch.GetEnabled(),
		})
	}

	if qh := resp.GetQuietHours(); qh != nil {
		prefs.QuietHours = pure.QuietHours{
			Enabled:   qh.GetEnabled(),
			StartTime: qh.GetStartTime(),
			EndTime:   qh.GetEndTime(),
			Timezone:  qh.GetTimezone(),
		}
	}

	return prefs, nil
}

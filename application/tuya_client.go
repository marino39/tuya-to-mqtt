package application

import "context"

type Device struct {
	UUID        string
	UID         string
	Name        string
	IP          string
	Sub         bool
	Model       string
	Category    string
	Online      bool
	ID          string
	TimeZone    string
	LocalKey    string
	UpdateTime  int
	ActiveTime  int
	OwnerID     string
	ProductID   string
	ProductName string
}

type TuyaClient interface {
	UserDevices(ctx context.Context, userId string) ([]*Device, error)
	DeviceStatus(ctx context.Context, deviceId string) ([]Status, error)
}

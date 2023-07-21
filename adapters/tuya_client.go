package adapters

import (
	"context"
	"fmt"
	"tuya-to-mqtt/application"

	"github.com/tuya/tuya-connector-go/connector"
)

type Response[T any] struct {
	Result PageCursor[T] `json:"result"`
}

type PageCursor[T any] struct {
	List []T `json:"list"`
}

type DeviceModel struct {
	UUID   string `json:"uuid"`
	UID    string `json:"uid"`
	Name   string `json:"name"`
	IP     string `json:"ip"`
	Sub    bool   `json:"sub"`
	Model  string `json:"model"`
	Status []struct {
		Code  string      `json:"code"`
		Value interface{} `json:"value"`
	} `json:"status"`
	Category    string `json:"category"`
	Online      bool   `json:"online"`
	ID          string `json:"id"`
	TimeZone    string `json:"time_zone"`
	LocalKey    string `json:"local_key"`
	UpdateTime  int    `json:"update_time"`
	ActiveTime  int    `json:"active_time"`
	OwnerID     string `json:"owner_id"`
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
}

type GetDeviceResponse struct {
	Code    int         `json:"code"`
	Msg     string      `json:"msg"`
	Success bool        `json:"success"`
	Result  DeviceModel `json:"result"`
	T       int64       `json:"t"`
}

type PostDeviceCmdResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
	Result  bool   `json:"result"`
	T       int64  `json:"t"`
}

type DeviceStatusResponse struct {
	Result []DeviceStatus `json:"result"`
}

type DeviceStatus struct {
	ID     string   `json:"id"`
	Status []Status `json:"status"`
}

type Status struct {
	Code  string `json:"code"`
	Value any    `json:"value"`
}

type TuyaClient struct {
}

func NewTuyaClient() *TuyaClient {
	return &TuyaClient{}
}

func (t *TuyaClient) UserDevices(ctx context.Context, userId string) ([]*application.Device, error) {
	var resp Response[DeviceModel]

	err := connector.MakeGetRequest(
		context.Background(),
		connector.WithAPIUri(fmt.Sprintf("/v1.3/iot-03/devices?source_type=tuyaUser&source_id=%s", userId)),
		connector.WithResp(&resp))
	if err != nil {
		return nil, err
	}

	return tuyaResponseDevicesToAppDevices(resp), nil
}

func (t *TuyaClient) DeviceStatus(ctx context.Context, deviceId string) ([]application.Status, error) {
	var respStat DeviceStatusResponse

	err := connector.MakeGetRequest(
		context.Background(),
		connector.WithAPIUri(fmt.Sprintf("/v1.0/iot-03/devices/status?device_ids=%s", deviceId)),
		connector.WithResp(&respStat))
	if err != nil {
		return nil, err
	}

	return tuyaResponseStatusToAppStatus(respStat), nil
}

var _ application.TuyaClient = &TuyaClient{}

func tuyaResponseDevicesToAppDevices(resp Response[DeviceModel]) []*application.Device {
	var devices []*application.Device
	for _, device := range resp.Result.List {
		devices = append(devices, &application.Device{
			UUID:        device.UUID,
			UID:         device.UID,
			Name:        device.Name,
			IP:          device.IP,
			Sub:         device.Sub,
			Model:       device.Model,
			Category:    device.Category,
			Online:      device.Online,
			ID:          device.ID,
			TimeZone:    device.TimeZone,
			LocalKey:    device.LocalKey,
			UpdateTime:  device.UpdateTime,
			ActiveTime:  device.ActiveTime,
			OwnerID:     device.OwnerID,
			ProductID:   device.ProductID,
			ProductName: device.ProductName,
		})
	}
	return devices
}

func tuyaResponseStatusToAppStatus(resp DeviceStatusResponse) []application.Status {
	var statusList []application.Status
	for _, device := range resp.Result {
		for _, status := range device.Status {
			statusList = append(statusList, application.Status{
				Code:  status.Code,
				Value: status.Value,
			})
		}
	}
	return statusList
}

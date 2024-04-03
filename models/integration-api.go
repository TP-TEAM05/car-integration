package models

const (
	TimestampFormat = "2006-01-02T15:04:05.999Z"
)

type IDatagram interface {
	GetIndex() int
	SetIndex(index int)

	GetType() string
	SetType(newType string)

	GetTimestamp() string
	SetTimestamp(timestamp string)
}

type INotifyDatagram interface {
	GetNotifyDatagram() *NotifyDatagram
	GetContent() interface{}
}

// BaseDatagram Datagram
type BaseDatagram struct {
	Index     int    `json:"index"`
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	// Checksum TODO
}

func (datagram *BaseDatagram) GetIndex() int {
	return datagram.Index
}

func (datagram *BaseDatagram) SetIndex(index int) {
	datagram.Index = index
}

func (datagram *BaseDatagram) GetType() string {
	return datagram.Type
}

func (datagram *BaseDatagram) SetType(newType string) {
	datagram.Type = newType
}

func (datagram *BaseDatagram) GetTimestamp() string {
	return datagram.Timestamp
}

func (datagram *BaseDatagram) SetTimestamp(timestamp string) {
	datagram.Timestamp = timestamp
}

type ConnectDatagram struct {
	BaseDatagram
}

type SubscribeDatagram struct {
	BaseDatagram
	Content  string  `json:"content"`
	Topic    string  `json:"topic"`
	Interval float32 `json:"interval"`
	Road     string  `json:"road"`
}

type UnsubscribeDatagram struct {
	BaseDatagram
	Content string `json:"content"`
}

type AcknowledgeDatagram struct {
	BaseDatagram
	AcknowledgingIndex int `json:"acknowledging_index"`
}

type KeepAliveDatagram struct {
	BaseDatagram
}

type PingDatagram struct {
	BaseDatagram
}

type RequestAreaDatagram struct {
	BaseDatagram
}

type AreaDatagram struct {
	BaseDatagram
	TopLeft     PositionJSON `json:"top_left"`
	BottomRight PositionJSON `json:"bottom_right"`
}

type NotifyDatagram struct {
	BaseDatagram
	VehicleId   int    `json:"vehicle_id"`
	Level       string `json:"level"`
	ContentType string `json:"content_type"`
}

func (notifyDatagram *NotifyDatagram) GetNotifyDatagram() *NotifyDatagram {
	return notifyDatagram
}

type GenericNotifyDatagram struct {
	NotifyDatagram
	Content GenericNotificationContent `json:"content"`
}

func (notifyDatagram *GenericNotifyDatagram) GetContent() interface{} {
	return notifyDatagram.Content
}

type HeadCollisionNotifyDatagram struct {
	NotifyDatagram
	Content HeadCollisionNotificationContent `json:"content"`
}

func (notifyDatagram *HeadCollisionNotifyDatagram) GetContent() interface{} {
	return notifyDatagram.Content
}

type ChainCollisionNotifyDatagram struct {
	NotifyDatagram
	Content ChainCollisionNotificationContent `json:"content"`
}

func (notifyDatagram *ChainCollisionNotifyDatagram) GetContent() interface{} {
	return notifyDatagram.Content
}

type CrossroadNotifyDatagram struct {
	NotifyDatagram
	Content CrossroadNotificationContent `json:"content"`
}

func (notifyDatagram *CrossroadNotifyDatagram) GetContent() interface{} {
	return notifyDatagram.Content
}

type NotifyVehicleDatagram struct {
	BaseDatagram
	Level       string `json:"level"`
	ContentType string `json:"content_type"`
}

type GenericNotifyVehicleDatagram struct {
	NotifyVehicleDatagram
	Content GenericNotificationContent `json:"content"`
}

type HeadCollisionNotifyVehicleDatagram struct {
	NotifyVehicleDatagram
	Content HeadCollisionNotificationContent `json:"content"`
}

type ChainCollisionNotifyVehicleDatagram struct {
	NotifyVehicleDatagram
	Content ChainCollisionNotificationContent `json:"content"`
}

type CrossroadNotifyVehicleDatagram struct {
	NotifyVehicleDatagram
	Content CrossroadNotificationContent `json:"content"`
}
type UpdateVehiclesDatagram struct {
	BaseDatagram
	Vehicles []UpdateVehicleVehicle `json:"vehicles"`
}

type UpdatePositionVehicleDatagram struct {
	BaseDatagram
	Vehicle UpdateVehicleVehicle `json:"vehicle"`
}

type ConnectVehicleDatagram struct {
	BaseDatagram
	Vin string `json:"vin"`
}

type DisconnectVehicleDatagram struct {
	BaseDatagram
	ConnectTo string `json:"connect_to"`
}

type UpdateVehicleDatagram struct {
	BaseDatagram
	Vehicle UpdateVehicleVehicle `json:"vehicle"`
}

type UpdateVehiclesVehicle struct {
	Timestamp       string  `json:"timestamp"`
	Id              int     `json:"id"`
	Longitude       float32 `json:"longitude"`
	Latitude        float32 `json:"latitude"`
	FrontUltrasonic float32 `json:"front_ultrasonic"`
	FrontLidar      float32 `json:"front_lidar"`
	SpeedFrontLeft  float32 `json:"speed_front_left"`
	SpeedFrontRight float32 `json:"speed_front_right"`
	SpeedRearLeft   float32 `json:"speed_rear_left"`
	SpeedRearRight  float32 `json:"speed_rear_right"`
}

type UpdateVehicleVehicle struct {
	Vin             string  `json:"vin"`
	Longitude       float32 `json:"longitude"`
	Latitude        float32 `json:"latitude"`
	FrontUltrasonic float32 `json:"front_ultrasonic"`
	FrontLidar      float32 `json:"front_lidar"`
	SpeedFrontLeft  float32 `json:"speed_front_left"`
	SpeedFrontRight float32 `json:"speed_front_right"`
	SpeedRearLeft   float32 `json:"speed_rear_left"`
	SpeedRearRight  float32 `json:"speed_rear_right"`
}

type UpdateVehicleDecision struct {
	Timestamp string `json:"timestamp"`
	Vin       string `json:"vin"`
}

type UpdateVehicleDecisionDatagram struct {
	BaseDatagram
	VehicleDecision UpdateVehicleDecision `json:"updateVehicleDecision"`
}

type UpdateNotificationsDatagram struct {
	BaseDatagram
	Notifications []UpdateNotificationsNotification `json:"notifications"`
}

type NetworkStatisticsDatagram struct {
	BaseDatagram
	NetworkStatistics []NetworkStatistics `json:"networkStatistics"`
}

type NetworkStatistics struct {
	PacketsReceived int64 `json:"packetsReceived"`
	ReceiveErrors   int64 `json:"receiveErrors"`
	AverageLatency  int64 `json:"averageLatency"`
	Jitter          int64 `json:"jitter"`
}

type UpdateNotificationsNotification struct {
	Timestamp   string      `json:"timestamp"`
	VehicleId   int         `json:"vehicle_id"`
	Level       string      `json:"level"`
	ContentType string      `json:"content_type"`
	Content     interface{} `json:"content"`
}

type PositionJSON struct {
	Lat float32 `json:"lat"`
	Lon float32 `json:"lon"`
}

type GenericNotificationContent struct {
	Text string `json:"text"`
}

type HeadCollisionNotificationContent struct {
	TargetVehicleId      int     `json:"target_vehicle_id"`
	TimeToCollision      float32 `json:"time_to_collision"`
	MaxSpeedExceededBy   float32 `json:"max_speed_exceeded_by"`
	BreakingDistanceDiff float32 `json:"breaking_distance_diff"`
}

type ChainCollisionNotificationContent struct {
	TargetVehicleId     int     `json:"target_vehicle_id"`
	CurrentDistance     float32 `json:"current_distance"`
	RecommendedDistance float32 `json:"recommended_distance"`
}

type CrossroadNotificationContent struct {
	Text       string `json:"text"`
	Order      int    `json:"order"`
	RightOfWay bool   `json:"right_of_way"`
}

package main

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
	Interval float32 `json:"interval"`
	Road     string  `json:"road"`
}

type UnsubscribeDatagram struct {
	BaseDatagram
	Content string `json:"content"`
}

type AcknowledgeDatagram struct {
	BaseDatagram
	AcknowledgingIndex int `json:"acknowledgingIndex"`
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
	TopLeft     PositionJSON `json:"topLeft"`
	BottomRight PositionJSON `json:"bottomRight"`
}

type NotifyDatagram struct {
	BaseDatagram
	VehicleId   int    `json:"vehicleId"`
	Level       string `json:"level"`
	ContentType string `json:"contentType"`
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
	ContentType string `json:"contentType"`
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
	Vehicles []UpdateVehiclesVehicle `json:"vehicles"`
}

type ConnectVehicleDatagram struct {
	BaseDatagram
	Vin string `json:"vin"`
}

type DisconnectVehicleDatagram struct {
	BaseDatagram
	ConnectTo string `json:"connectTo"`
}

type UpdateVehicleDatagram struct {
	BaseDatagram
	Vehicle UpdateVehicleVehicle `json:"vehicle"`
}

type UpdateVehiclesVehicle struct {
	Timestamp    string       `json:"timestamp"`
	Id           int          `json:"id"`
	Type         string       `json:"type"`
	Speed        float32      `json:"speed"`
	Acceleration float32      `json:"acceleration"`
	Heading      float32      `json:"heading"`
	Position     PositionJSON `json:"position"`
	LaneId       string       `json:"laneId"`
}

type UpdateVehicleVehicle struct {
	Vin          string       `json:"vin"`
	Speed        float32      `json:"speed"`
	Acceleration float32      `json:"acceleration"`
	Heading      float32      `json:"heading"`
	Position     PositionJSON `json:"position"`
	LaneId       string       `json:"laneId"`
}

type UpdateNotificationsDatagram struct {
	BaseDatagram
	Notifications []UpdateNotificationsNotification `json:"notifications"`
}

type UpdateNotificationsNotification struct {
	Timestamp   string      `json:"timestamp"`
	VehicleId   int         `json:"vehicleId"`
	Level       string      `json:"level"`
	ContentType string      `json:"contentType"`
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
	TargetVehicleId      int     `json:"targetVehicleId"`
	TimeToCollision      float32 `json:"timeToCollision"`
	MaxSpeedExceededBy   float32 `json:"maxSpeedExceededBy"`
	BreakingDistanceDiff float32 `json:"breakingDistanceDiff"`
}

type ChainCollisionNotificationContent struct {
	TargetVehicleId     int     `json:"targetVehicleId"`
	CurrentDistance     float32 `json:"currentDistance"`
	RecommendedDistance float32 `json:"recommendedDistance"`
}

type CrossroadNotificationContent struct {
	Text       string `json:"text"`
	Order      int    `json:"order"`
	RightOfWay bool   `json:"rightOfWay"`
}

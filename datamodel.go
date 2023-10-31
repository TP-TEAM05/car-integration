package main

import (
	"fmt"
	"sync"
	"time"

	api "github.com/ReCoFIIT/traffic-dt-integration-module-api"
)

// For now

func ParseTime(timestamp string) time.Time {
	t, err := time.Parse(api.TimestampFormat, timestamp)
	if err != nil {
		fmt.Printf("Failed to parse timestamp %v\n", timestamp)
	}
	return t
}

type DataModel struct {
	sync.Mutex
	Area                   *Area
	Vehicles               map[string]*Vehicle
	NextVehicleId          int
	VehicleConnectionsById map[int]*VehicleConnection // Maps vehicleId to its connection
	Notifications          map[int]map[string]*Notification
	NotificationDuration   float32
	NextNotificationId     int
}

func NewDataModel(area *Area, notificationDuration float32) *DataModel {
	return &DataModel{
		Area:                   area,
		Vehicles:               make(map[string]*Vehicle),
		Notifications:          make(map[int]map[string]*Notification),
		VehicleConnectionsById: make(map[int]*VehicleConnection),
		NotificationDuration:   notificationDuration,
	}
}

func (dataModel *DataModel) AddNotification(datagram api.INotifyDatagram, safe bool) {
	if safe {
		dataModel.Lock()
		defer dataModel.Unlock()
	}

	notifyDiagram := datagram.GetNotifyDatagram()
	notificationId := dataModel.NextNotificationId
	dataModel.NextNotificationId++

	vehicleNotificationsMap, ok := dataModel.Notifications[notifyDiagram.VehicleId]
	if !ok {
		vehicleNotificationsMap = make(map[string]*Notification)
		dataModel.Notifications[notifyDiagram.VehicleId] = vehicleNotificationsMap
	}

	// Prepare new notification
	notification := &Notification{
		Id: notificationId,
		Datagram: &api.UpdateNotificationsNotification{
			Timestamp:   notifyDiagram.Timestamp,
			VehicleId:   notifyDiagram.VehicleId,
			Level:       notifyDiagram.Level,
			ContentType: notifyDiagram.ContentType,
			Content:     datagram.GetContent(),
		},
	}

	// Only add if newer than the one that potentially exists and also only if the danger is bigger (or the vehicle id is the same)
	existingNotification, ok := vehicleNotificationsMap[notifyDiagram.ContentType]
	if ok {
		replaceable, err := existingNotification.ReplaceableBy(notification)
		if err != nil {
			fmt.Printf("Error when checking notification replaceability %v\n", err)
			return
		}
		if !replaceable {
			return
		}
	}

	vehicleNotificationsMap[notifyDiagram.ContentType] = notification

	// Delete notification after some time
	time.AfterFunc(time.Duration(float32(time.Second)*dataModel.NotificationDuration), func() {
		dataModel.DeleteNotification(notifyDiagram.VehicleId, notifyDiagram.ContentType, notificationId, true)
	})
}

func (dataModel *DataModel) DeleteNotification(vehicleId int, contentType string, notificationId int, safe bool) {
	if safe {
		dataModel.Lock()
		defer dataModel.Unlock()
	}
	vehicleNotificationsMap, ok := dataModel.Notifications[vehicleId]
	if ok {
		existingNotification, ok := vehicleNotificationsMap[contentType]
		if ok && existingNotification.Id == notificationId {
			delete(vehicleNotificationsMap, contentType)
		}
	}
}

func (dataModel *DataModel) GetNotifications(safe bool) []api.UpdateNotificationsNotification {
	if safe {
		dataModel.Lock()
		defer dataModel.Unlock()
	}

	var notifications = make([]api.UpdateNotificationsNotification, dataModel.GetNotificationsCount(false))
	i := 0
	for _, vehicleNotifications := range dataModel.Notifications {
		for _, notification := range vehicleNotifications {
			notifications[i] = *notification.Datagram
			i++
		}
	}
	return notifications
}

// Returns count of items of nested map (one level deep). */
func (dataModel *DataModel) GetNotificationsCount(safe bool) int {
	if safe {
		dataModel.Lock()
		defer dataModel.Unlock()
	}
	count := 0
	for _, innerMap := range dataModel.Notifications {
		count += len(innerMap)
	}
	return count
}

func (dataModel *DataModel) UpdateVehicle(connection *VehicleConnection, datagram *api.UpdateVehicleDatagram, safe bool) {
	if safe {
		dataModel.Lock()
		defer dataModel.Unlock()
	}

	vehicle := datagram.Vehicle

	savedVehicle, ok := dataModel.Vehicles[vehicle.Vin]
	if !ok {
		savedVehicle = &Vehicle{
			Id:   dataModel.NextVehicleId,
			Vin:  vehicle.Vin,
			Type: GetVehicleType(vehicle.Vin),
		}
		dataModel.NextVehicleId++
		dataModel.Vehicles[vehicle.Vin] = savedVehicle
	} else {
		newTime, err := time.Parse(api.TimestampFormat, datagram.Timestamp)
		if err != nil {
			fmt.Printf("Failed to parse %v\n", datagram.Timestamp)
			return
		}

		lastTime, err := time.Parse(api.TimestampFormat, savedVehicle.Timestamp)
		if err != nil {
			fmt.Printf("Failed to parse %v\n", savedVehicle.Timestamp)
			return
		}

		// We want to discard the received datagram if it was older than current data we have
		if newTime.Before(lastTime) {
			return
		}
	}

	savedVehicle.Timestamp = datagram.Timestamp
	savedVehicle.Speed = vehicle.Speed
	savedVehicle.Acceleration = vehicle.Acceleration
	savedVehicle.Heading = vehicle.Heading
	savedVehicle.Position = vehicle.Position
	savedVehicle.LaneId = vehicle.LaneId

	dataModel.VehicleConnectionsById[savedVehicle.Id] = connection
}

// DeleteVehicle removes the vehicle identified by the vin number from the DataModel.
func (dataModel *DataModel) DeleteVehicle(vin string, safe bool) {
	if safe {
		dataModel.Lock()
		defer dataModel.Unlock()
	}
	delete(dataModel.Vehicles, vin)
}

func (dataModel *DataModel) GetVehicles(safe bool) []api.UpdateVehiclesVehicle {
	if safe {
		dataModel.Lock()
		defer dataModel.Unlock()
	}

	var vehicles = make([]api.UpdateVehiclesVehicle, len(dataModel.Vehicles))
	i := 0
	for _, vehicle := range dataModel.Vehicles {
		vehicles[i] = api.UpdateVehiclesVehicle{
			Id:           vehicle.Id,
			Timestamp:    vehicle.Timestamp,
			Type:         vehicle.Type,
			Speed:        vehicle.Speed,
			Acceleration: vehicle.Acceleration,
			Heading:      vehicle.Heading,
			Position:     vehicle.Position,
			LaneId:       vehicle.LaneId,
		}
		i++
	}
	return vehicles
}

func (dataModel *DataModel) GetVehicleConnection(vehicleId int, safe bool) *VehicleConnection {
	if safe {
		dataModel.Lock()
		defer dataModel.Unlock()
	}

	if connection, ok := dataModel.VehicleConnectionsById[vehicleId]; ok {
		return connection
	}
	return nil
}

// GetVehicleType Currently we support only cars and don't parse vin number.
func GetVehicleType(vin string) string {
	return "car"
}

type Vehicle struct {
	Timestamp    string
	Id           int
	Vin          string
	Type         string
	Speed        float32
	Acceleration float32
	Heading      float32
	Position     api.PositionJSON
	LaneId       string
}

type Notification struct {
	Id       int
	Datagram *api.UpdateNotificationsNotification
}

// Returns true if other Notification should replace this notification in the means of importance. Note this can only
// be called on notifications of same contentType
func (notification *Notification) ReplaceableBy(other *Notification) (bool, error) {
	if notification.Datagram.ContentType != other.Datagram.ContentType {
		return false, fmt.Errorf("other and notification ContentType mismatch")
	}

	// Other is older don't replace
	existingTime := ParseTime(notification.Datagram.Timestamp)
	otherTime := ParseTime(other.Datagram.Timestamp)
	if existingTime.After(otherTime) {
		return false, nil
	}

	// Check whether other has higher importance level
	notificationLevelValues := map[string]int{
		"info":    0,
		"warning": 1,
		"danger":  2,
	}
	otherHasHigherLevel := notificationLevelValues[other.Datagram.Level] >= notificationLevelValues[notification.Datagram.Level]
	if otherHasHigherLevel {
		return true, nil
	}

	// We can discard outdated notification with higher level if the targetVehicleIsTheSame
	switch notification.Datagram.ContentType {
	case "head_collision":
		targetVehicleId := notification.Datagram.Content.(api.HeadCollisionNotificationContent).TargetVehicleId
		otherTargetVehicleId := other.Datagram.Content.(api.HeadCollisionNotificationContent).TargetVehicleId
		if targetVehicleId == otherTargetVehicleId {
			return true, nil
		}
	case "chain_collision":
		targetVehicleId := notification.Datagram.Content.(api.ChainCollisionNotificationContent).TargetVehicleId
		otherTargetVehicleId := other.Datagram.Content.(api.ChainCollisionNotificationContent).TargetVehicleId
		if targetVehicleId == otherTargetVehicleId {
			return true, nil
		}
	}

	return false, nil
}

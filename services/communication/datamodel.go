package communication

import (
	"car-integration/models"
	"fmt"
	"sync"
	"time"

	api "github.com/ReCoFIIT/integration-api"
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
	Area                   *models.Area
	Vehicles               map[string]*Vehicle
	VehicleDecisions       map[string]*api.UpdateVehicleDecision
	NextVehicleId          int
	VehicleConnectionsById map[int]*VehicleConnection // Maps vehicleId to its connection
	Notifications          map[int]map[string]*Notification
	NotificationDuration   float32
	NextNotificationId     int

	updateCond                *sync.Cond
	updateCondDecision        *sync.Cond
	UpdatedVehicleVin         string
	UpdatedVehicleDecisionVin string
}

func NewDataModel(area *models.Area, notificationDuration float32) *DataModel {
	dm := &DataModel{
		Area:                   area,
		Vehicles:               make(map[string]*Vehicle),
		VehicleDecisions:       make(map[string]*api.UpdateVehicleDecision),
		Notifications:          make(map[int]map[string]*Notification),
		VehicleConnectionsById: make(map[int]*VehicleConnection),
		NotificationDuration:   notificationDuration,
	}
	dm.updateCond = sync.NewCond(&dm.Mutex)
	dm.updateCondDecision = sync.NewCond(&dm.Mutex)
	return dm
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
			Timestamp:            datagram.Timestamp,
			UpdateVehicleVehicle: vehicle,
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

	savedVehicle.UpdateVehicleVehicle = vehicle

	dataModel.VehicleConnectionsById[savedVehicle.Id] = connection
	dataModel.UpdatedVehicleVin = vehicle.Vin
	dataModel.updateCond.Broadcast()
}

func (dataModel *DataModel) UpdateVehicleDecision(connection *ProcessorConnection, datagram *api.UpdateVehicleDecisionDatagram, safe bool) {
	if safe {
		dataModel.Lock()
		defer dataModel.Unlock()
	}

	vehicleDecision := datagram.VehicleDecision

	savedVehicle, ok := dataModel.VehicleDecisions[vehicleDecision.Vin]
	if !ok {
		savedVehicle = &api.UpdateVehicleDecision{
			Message: vehicleDecision.Message,
			Vin:     vehicleDecision.Vin,
		}
		dataModel.VehicleDecisions[savedVehicle.Vin] = savedVehicle
	} else {
		newTime, err := time.Parse(api.TimestampFormat, datagram.BaseDatagram.Timestamp)
		if err != nil {
			fmt.Printf("Failed to parse %v\n", datagram.BaseDatagram.Timestamp)
			return
		}

		lastTime, err := time.Parse(api.TimestampFormat, datagram.BaseDatagram.Timestamp)
		if err != nil {
			fmt.Printf("Failed to parse %v\n", datagram.BaseDatagram.Timestamp)
			return
		}

		// We want to discard the received datagram if it was older than current data we have
		if newTime.Before(lastTime) {
			return
		}
	}

	dataModel.UpdatedVehicleDecisionVin = savedVehicle.Vin

	dataModel.updateCondDecision.Broadcast()
}

// DeleteVehicle removes the vehicle identified by the vin number from the DataModel.
func (dataModel *DataModel) DeleteVehicle(vin string, safe bool) {
	if safe {
		dataModel.Lock()
		defer dataModel.Unlock()
	}
	delete(dataModel.Vehicles, vin)
}

func (dataModel *DataModel) GetVehicles(safe bool) []api.UpdateVehicleVehicle {
	if safe {
		dataModel.Lock()
		defer dataModel.Unlock()
	}

	var vehicles = make([]api.UpdateVehicleVehicle, len(dataModel.Vehicles))
	i := 0
	for _, vehicle := range dataModel.Vehicles {
		vehicles[i] = api.UpdateVehicleVehicle{
			Vin:             vehicle.Vin,
			Longitude:       vehicle.Longitude,
			Latitude:        vehicle.Latitude,
			FrontUltrasonic: vehicle.FrontUltrasonic,
			FrontLidar:      vehicle.FrontLidar,
			SpeedFrontLeft:  vehicle.SpeedFrontLeft,
			SpeedFrontRight: vehicle.SpeedFrontRight,
			SpeedRearRight:  vehicle.SpeedRearRight,
			SpeedRearLeft:   vehicle.SpeedRearLeft,
		}
		i++
	}
	return vehicles
}

func (dataModel *DataModel) GetVehicleById(id string) api.UpdateVehicleVehicle {
	// Look up the vehicle by ID directly
	vehicle, ok := dataModel.Vehicles[id]
	if !ok {
		// Vehicle not found

	}

	// Vehicle found, return the corresponding UpdateVehiclesVehicle
	return api.UpdateVehicleVehicle{
		Vin:             vehicle.Vin,
		Longitude:       vehicle.Longitude,
		Latitude:        vehicle.Latitude,
		FrontUltrasonic: vehicle.FrontUltrasonic,
		FrontLidar:      vehicle.FrontLidar,
		SpeedFrontLeft:  vehicle.SpeedFrontLeft,
		SpeedFrontRight: vehicle.SpeedFrontRight,
		SpeedRearRight:  vehicle.SpeedRearRight,
		SpeedRearLeft:   vehicle.SpeedRearLeft,
	}
}

func (dataModel *DataModel) GetVehicleDecisionById(id string) api.UpdateVehicleDecision {
	// Look up the vehicle by ID directly
	vehicle, ok := dataModel.VehicleDecisions[id]
	if !ok {
		// Vehicle not found

	}
	// Vehicle found, return the corresponding UpdateVehiclesVehicle
	return api.UpdateVehicleDecision{
		Vin:     vehicle.Vin,
		Message: vehicle.Message,
	}
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

type Vehicle struct {
	api.UpdateVehicleVehicle
	Timestamp string
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

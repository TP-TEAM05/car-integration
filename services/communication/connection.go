package communication

import (
	"car-integration/services/redis"
	"car-integration/services/statistics"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	api "github.com/ReCoFIIT/integration-api"
)

type IConnection interface {
	WriteDatagram(datagram api.IDatagram, safe bool)
	ProcessDatagram(data []byte, safe bool)
	OnDead(safe bool) // Called when the KeepAliveTimeout is reached before deletion of this connection.
	GetKeepAliveTimeout(safe bool) float32
	GetClientAddress(safe bool) *net.UDPAddr

	SetKeepAliveTimer(timer *time.Timer, safe bool)
	GetKeepAliveTimer(safe bool) *time.Timer
}

/* Common Connection */

type Connection struct {
	sync.Mutex
	UDPConn           *net.UDPConn
	ClientAddress     *net.UDPAddr
	NextSendIndex     int
	LastReceivedIndex int
	DataModel         *DataModel
	KeepAliveTimeout  float32 // Seconds, after which is the connection discarded if no datagram arrived. 0 for no timeout
	KeepAliveTimer    *time.Timer
}

func (connection *Connection) WriteDatagram(datagram api.IDatagram, safe bool) {
	if safe {
		connection.Lock()
		defer connection.Unlock()
	}

	datagram.SetTimestamp(time.Now().UTC().Format(api.TimestampFormat))
	datagram.SetIndex(connection.NextSendIndex)
	connection.NextSendIndex++

	data, err := json.Marshal(datagram)
	if err != nil {
		fmt.Printf("Error marshalling datagram %v with error %v\n", datagram, err)
		return
	}

	_, err = connection.UDPConn.WriteToUDP(data, connection.ClientAddress)
	if err != nil {
		fmt.Printf("Error writing datagram with error %v\n", err)
		return
	}
	// fmt.Printf("Sending message to %v: %s\n", connection.ClientAddress, data[:min(len(data), 128)])
}

func (connection *Connection) OnDead(safe bool) {
}

func (connection *Connection) GetKeepAliveTimeout(safe bool) float32 {
	if safe {
		connection.Lock()
		defer connection.Unlock()
	}
	return connection.KeepAliveTimeout
}

func (connection *Connection) GetClientAddress(safe bool) *net.UDPAddr {
	if safe {
		connection.Lock()
		defer connection.Unlock()
	}
	return connection.ClientAddress
}

func (connection *Connection) GetKeepAliveTimer(safe bool) *time.Timer {
	if safe {
		connection.Lock()
		defer connection.Unlock()
	}
	return connection.KeepAliveTimer
}

func (connection *Connection) SetKeepAliveTimer(timer *time.Timer, safe bool) {
	if safe {
		connection.Lock()
		defer connection.Unlock()
	}
	connection.KeepAliveTimer = timer
}

/* Connection from Processor */

type ProcessorConnection struct {
	Connection
	Subscriptions map[string]*Subscription // Mapping content to subscription (only one subscription to each type can exist)
}

func (connection *ProcessorConnection) ProcessDatagram(data []byte, safe bool) {
	// Parse data to JSON
	var datagram api.BaseDatagram
	err := json.Unmarshal(data, &datagram)
	if err != nil {
		fmt.Print("Parsing JSON failed.")
		return
	}
	// TODO uncomment this
	//if datagram.Index <= connection.LastReceivedIndex {
	//	return
	//}

	switch datagram.Type {
	case "connect":
		var connectDatagram api.ConnectDatagram
		_ = json.Unmarshal(data, &connectDatagram)
		response := &api.AcknowledgeDatagram{
			BaseDatagram:       api.BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: connectDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "subscribe":
		var subscribeDatagram api.SubscribeDatagram
		_ = json.Unmarshal(data, &subscribeDatagram)

		// Create subscription
		connection.Subscribe(&subscribeDatagram, safe)

		// Send acknowledgement
		response := &api.AcknowledgeDatagram{
			BaseDatagram:       api.BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: subscribeDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "unsubscribe":
		var unsubscribeDatagram api.UnsubscribeDatagram
		_ = json.Unmarshal(data, &unsubscribeDatagram)

		// Delete subscription
		connection.Unsubscribe(unsubscribeDatagram.Content, safe)

		// Send acknowledgement
		response := &api.AcknowledgeDatagram{
			BaseDatagram:       api.BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: unsubscribeDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "keepalive":
		var keepAliveDatagram api.KeepAliveDatagram
		_ = json.Unmarshal(data, &keepAliveDatagram)
		response := &api.AcknowledgeDatagram{
			BaseDatagram:       api.BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: keepAliveDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "ping":
		var pingDatagram api.KeepAliveDatagram
		_ = json.Unmarshal(data, &pingDatagram)
		response := &api.AcknowledgeDatagram{
			BaseDatagram:       api.BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: pingDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "request_area":
		var requestAreaDatagram api.RequestAreaDatagram
		_ = json.Unmarshal(data, &requestAreaDatagram)

		response := &api.AreaDatagram{
			BaseDatagram: api.BaseDatagram{Type: "area"},
			TopLeft:      connection.DataModel.Area.TopLeft,
			BottomRight:  connection.DataModel.Area.BottomRight,
		}
		connection.WriteDatagram(response, safe)

	case "decision_update":
		var decisionUpdateDatagram api.UpdateVehicleDecisionDatagram
		_ = json.Unmarshal(data, &decisionUpdateDatagram)
		fmt.Printf("decision update arrived....\n")

		connection.DataModel.UpdateVehicleDecision(connection, &decisionUpdateDatagram, true)

	case "notify":
		var notifyDatagram api.NotifyDatagram
		_ = json.Unmarshal(data, &notifyDatagram)
		response := &api.AcknowledgeDatagram{
			BaseDatagram:       api.BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: notifyDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

		var specificNotifyDatagram api.INotifyDatagram

		switch notifyDatagram.ContentType {
		case "generic":
			var genericDatagram api.GenericNotifyDatagram
			_ = json.Unmarshal(data, &genericDatagram)
			specificNotifyDatagram = &genericDatagram
		case "head_collision":
			var headCollisionDatagram api.HeadCollisionNotifyDatagram
			_ = json.Unmarshal(data, &headCollisionDatagram)
			specificNotifyDatagram = &headCollisionDatagram
		case "chain_collision":
			var chainCollisionDatagram api.ChainCollisionNotifyDatagram
			_ = json.Unmarshal(data, &chainCollisionDatagram)
			specificNotifyDatagram = &chainCollisionDatagram
		case "crossroad":
			var crossroadDatagram api.CrossroadNotifyDatagram
			_ = json.Unmarshal(data, &crossroadDatagram)
			specificNotifyDatagram = &crossroadDatagram
		}

		// Save notification
		connection.DataModel.AddNotification(specificNotifyDatagram, true)

		// Send notification to target vehicle
		vehicleConnection := connection.DataModel.GetVehicleConnection(notifyDatagram.VehicleId, true)
		if vehicleConnection != nil {
			var specificNotifyVehicleDatagram api.IDatagram
			notifyVehicleDatagram := &api.NotifyVehicleDatagram{
				BaseDatagram: api.BaseDatagram{Type: "notify_vehicle"},
				Level:        notifyDatagram.Level,
				ContentType:  notifyDatagram.ContentType,
			}
			switch notifyDatagram.ContentType {
			case "generic":
				specificNotifyVehicleDatagram = &api.GenericNotifyVehicleDatagram{
					NotifyVehicleDatagram: *notifyVehicleDatagram,
					Content:               specificNotifyDatagram.GetContent().(api.GenericNotificationContent),
				}
			case "head_collision":
				specificNotifyVehicleDatagram = &api.HeadCollisionNotifyVehicleDatagram{
					NotifyVehicleDatagram: *notifyVehicleDatagram,
					Content:               specificNotifyDatagram.GetContent().(api.HeadCollisionNotificationContent),
				}
			case "chain_collision":
				specificNotifyVehicleDatagram = &api.ChainCollisionNotifyVehicleDatagram{
					NotifyVehicleDatagram: *notifyVehicleDatagram,
					Content:               specificNotifyDatagram.GetContent().(api.ChainCollisionNotificationContent),
				}
			case "crossroad":
				specificNotifyVehicleDatagram = &api.CrossroadNotifyVehicleDatagram{
					NotifyVehicleDatagram: *notifyVehicleDatagram,
					Content:               specificNotifyDatagram.GetContent().(api.CrossroadNotificationContent),
				}
			}

			vehicleConnection.WriteDatagram(specificNotifyVehicleDatagram, true)
		}
	}

	if safe {
		connection.Lock()
	}
	connection.LastReceivedIndex = datagram.Index
	if safe {
		connection.Unlock()
	}
}

func (connection *ProcessorConnection) Subscribe(datagram *api.SubscribeDatagram, safe bool) {
	if safe {
		connection.Lock()
		defer connection.Unlock()
	}
	connection.Unsubscribe(datagram.Content, false) // Delete existing subscription if any
	subscription := &Subscription{
		&connection.Connection,
		datagram.Content,
		datagram.Topic,
		datagram.Interval,
		make(chan bool),
	}
	connection.Subscriptions[datagram.Content] = subscription
	go func() {
		err := subscription.Start()
		if err != nil {
			fmt.Printf("Subscription ended due to an error: %v\n", err)
		}
	}()
}

func (connection *ProcessorConnection) Unsubscribe(content string, safe bool) {
	if safe {
		connection.Lock()
		defer connection.Unlock()
	}
	subscription, ok := connection.Subscriptions[content]
	if ok {
		go func() { subscription.Stop() }() // We have to call this in own coroutine because it may block, and would hold the connection lock
		delete(connection.Subscriptions, content)
	}
}

func (connection *ProcessorConnection) UnsubscribeAll(safe bool) {
	if safe {
		connection.Lock()
		defer connection.Unlock()
	}
	for content := range connection.Subscriptions {
		connection.Unsubscribe(content, false)
	}
}

func (connection *ProcessorConnection) OnDead(safe bool) {
	connection.UnsubscribeAll(safe)
}

/* Connection from Vehicle */

type VehicleConnection struct {
	Connection
	VinNumber    string
	Subscription *Subscription
	NetworkStats *statistics.NetworkStatistics
}

func (connection *VehicleConnection) Subscribe(safe bool) {
	if safe {
		connection.Lock()
		defer connection.Unlock()
	}
	subscription := &Subscription{
		&connection.Connection,
		"decision-update",
		connection.VinNumber,
		1,
		make(chan bool),
	}

	connection.Subscription = subscription

	go func() {
		err := subscription.Start()
		if err != nil {
			fmt.Printf("Subscription ended due to an error: %v\n", err)
		}
	}()
}

func (connection *VehicleConnection) ProcessDatagram(data []byte, safe bool) {

	// Parse data to JSON
	var datagram api.BaseDatagram
	err := json.Unmarshal(data, &datagram)
	if err != nil {
		fmt.Print("Parsing JSON failed: ", err)
		return
	}
	// TODO uncomment this
	//if datagram.Index <= connection.LastReceivedIndex {
	//	return
	//}

	switch datagram.Type {
	case "ping":
		var pingDatagram api.KeepAliveDatagram
		_ = json.Unmarshal(data, &pingDatagram)
		response := &api.AcknowledgeDatagram{
			BaseDatagram:       api.BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: pingDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "update_vehicle":
		var updateVehicleDatagram api.UpdateVehicleDatagram
		// DEBUG: Here are the data received from vehicle

		_ = json.Unmarshal(data, &updateVehicleDatagram)

		// TODO: Generalize this nasty hack for remapping key names

		type VehicleData struct {
			Vehicle struct {
				Hacc float32 `json:"hacc"`
			} `json:"vehicle"`
		}

		remap := VehicleData{}

		_ = json.Unmarshal(data, &remap)

		updateVehicleDatagram.Vehicle.GpsHorizontalAccuracy = remap.Vehicle.Hacc

		// Continue with the rest of the parsing

		// Update vehicle data in connection
		if safe {
			connection.Lock()
		}
		connection.VinNumber = updateVehicleDatagram.Vehicle.Vin
		if safe {
			connection.Unlock()
		}

		connection.NetworkStats.Update(updateVehicleDatagram, time.Now().UTC())
		// Save stats to Redis
		err := redis.SaveNetworkStats(updateVehicleDatagram.Vehicle.Vin, &connection.NetworkStats.Stats)
		if err != nil {
			fmt.Println("Failed to save network stats:", err)
		}

		// Create subscription
		if connection.Subscription == nil {
			fmt.Printf("Subscribe function call..." + connection.VinNumber + "\n")
			connection.Subscribe(safe)
		}

		// connection.DataModel.Lock()
		// insideArea := connection.DataModel.Area.Contains(&updateVehicleDatagram.Vehicle.Position)
		// connection.DataModel.Unlock()

		if true {
			connection.DataModel.UpdateVehicle(connection, &updateVehicleDatagram, true)
		} else {
			// Disconnect vehicle which is outside the managed area
			connection.DataModel.DeleteVehicle(updateVehicleDatagram.Vehicle.Vin, true)
			response := &api.DisconnectVehicleDatagram{
				BaseDatagram: api.BaseDatagram{Type: "disconnect_vehicle"},
				ConnectTo:    "NOT IMPLEMENTED", // Should contain connection string to the following Integration Module, out of scope for now

			}
			connection.WriteDatagram(response, safe)
		}
	}

	if safe {
		connection.Lock()
	}
	connection.LastReceivedIndex = datagram.Index
	if safe {
		connection.Unlock()
	}
}

func (connection *VehicleConnection) OnDead(safe bool) {
	connection.DataModel.DeleteVehicle(connection.VinNumber, true)
}

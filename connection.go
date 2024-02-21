package main

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

type IConnection interface {
	WriteDatagram(datagram IDatagram, safe bool)
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

func (connection *Connection) WriteDatagram(datagram IDatagram, safe bool) {
	if safe {
		connection.Lock()
		defer connection.Unlock()
	}

	datagram.SetTimestamp(time.Now().UTC().Format(TimestampFormat))
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
	fmt.Printf("Sending message to %v: %s\n", connection.ClientAddress, data[:min(len(data), 128)])
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
	var datagram BaseDatagram
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
		var connectDatagram ConnectDatagram
		_ = json.Unmarshal(data, &connectDatagram)
		response := &AcknowledgeDatagram{
			BaseDatagram:       BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: connectDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "subscribe":
		var subscribeDatagram SubscribeDatagram
		_ = json.Unmarshal(data, &subscribeDatagram)

		// Create subscription
		connection.Subscribe(&subscribeDatagram, safe)

		// Send acknowledgement
		response := &AcknowledgeDatagram{
			BaseDatagram:       BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: subscribeDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "unsubscribe":
		var unsubscribeDatagram UnsubscribeDatagram
		_ = json.Unmarshal(data, &unsubscribeDatagram)

		// Delete subscription
		connection.Unsubscribe(unsubscribeDatagram.Content, safe)

		// Send acknowledgement
		response := &AcknowledgeDatagram{
			BaseDatagram:       BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: unsubscribeDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "keepalive":
		var keepAliveDatagram KeepAliveDatagram
		_ = json.Unmarshal(data, &keepAliveDatagram)
		response := &AcknowledgeDatagram{
			BaseDatagram:       BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: keepAliveDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "ping":
		var pingDatagram KeepAliveDatagram
		_ = json.Unmarshal(data, &pingDatagram)
		response := &AcknowledgeDatagram{
			BaseDatagram:       BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: pingDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "request_area":
		var requestAreaDatagram RequestAreaDatagram
		_ = json.Unmarshal(data, &requestAreaDatagram)

		response := &AreaDatagram{
			BaseDatagram: BaseDatagram{Type: "area"},
			TopLeft:      connection.DataModel.Area.TopLeft,
			BottomRight:  connection.DataModel.Area.BottomRight,
		}
		connection.WriteDatagram(response, safe)

	case "notify":
		var notifyDatagram NotifyDatagram
		_ = json.Unmarshal(data, &notifyDatagram)
		response := &AcknowledgeDatagram{
			BaseDatagram:       BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: notifyDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

		var specificNotifyDatagram INotifyDatagram

		switch notifyDatagram.ContentType {
		case "generic":
			var genericDatagram GenericNotifyDatagram
			_ = json.Unmarshal(data, &genericDatagram)
			specificNotifyDatagram = &genericDatagram
		case "head_collision":
			var headCollisionDatagram HeadCollisionNotifyDatagram
			_ = json.Unmarshal(data, &headCollisionDatagram)
			specificNotifyDatagram = &headCollisionDatagram
		case "chain_collision":
			var chainCollisionDatagram ChainCollisionNotifyDatagram
			_ = json.Unmarshal(data, &chainCollisionDatagram)
			specificNotifyDatagram = &chainCollisionDatagram
		case "crossroad":
			var crossroadDatagram CrossroadNotifyDatagram
			_ = json.Unmarshal(data, &crossroadDatagram)
			specificNotifyDatagram = &crossroadDatagram
		}

		// Save notification
		connection.DataModel.AddNotification(specificNotifyDatagram, true)

		// Send notification to target vehicle
		vehicleConnection := connection.DataModel.GetVehicleConnection(notifyDatagram.VehicleId, true)
		if vehicleConnection != nil {
			var specificNotifyVehicleDatagram IDatagram
			notifyVehicleDatagram := &NotifyVehicleDatagram{
				BaseDatagram: BaseDatagram{Type: "notify_vehicle"},
				Level:        notifyDatagram.Level,
				ContentType:  notifyDatagram.ContentType,
			}
			switch notifyDatagram.ContentType {
			case "generic":
				specificNotifyVehicleDatagram = &GenericNotifyVehicleDatagram{
					NotifyVehicleDatagram: *notifyVehicleDatagram,
					Content:               specificNotifyDatagram.GetContent().(GenericNotificationContent),
				}
			case "head_collision":
				specificNotifyVehicleDatagram = &HeadCollisionNotifyVehicleDatagram{
					NotifyVehicleDatagram: *notifyVehicleDatagram,
					Content:               specificNotifyDatagram.GetContent().(HeadCollisionNotificationContent),
				}
			case "chain_collision":
				specificNotifyVehicleDatagram = &ChainCollisionNotifyVehicleDatagram{
					NotifyVehicleDatagram: *notifyVehicleDatagram,
					Content:               specificNotifyDatagram.GetContent().(ChainCollisionNotificationContent),
				}
			case "crossroad":
				specificNotifyVehicleDatagram = &CrossroadNotifyVehicleDatagram{
					NotifyVehicleDatagram: *notifyVehicleDatagram,
					Content:               specificNotifyDatagram.GetContent().(CrossroadNotificationContent),
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

func (connection *ProcessorConnection) Subscribe(datagram *SubscribeDatagram, safe bool) {
	if safe {
		connection.Lock()
		defer connection.Unlock()
	}
	connection.Unsubscribe(datagram.Content, false) // Delete existing subscription if any
	subscription := &Subscription{
		connection,
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
	VinNumber string
}

func (connection *VehicleConnection) ProcessDatagram(data []byte, safe bool) {

	// Parse data to JSON
	var datagram BaseDatagram
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
		var pingDatagram KeepAliveDatagram
		_ = json.Unmarshal(data, &pingDatagram)
		response := &AcknowledgeDatagram{
			BaseDatagram:       BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: pingDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "update_vehicle":
		var updateVehicleDatagram UpdateVehicleDatagram
		_ = json.Unmarshal(data, &updateVehicleDatagram)

		// Update vehicle data in connection
		if safe {
			connection.Lock()
		}
		connection.VinNumber = updateVehicleDatagram.Vehicle.Vin
		if safe {
			connection.Unlock()
		}

		// connection.DataModel.Lock()
		// insideArea := connection.DataModel.Area.Contains(&updateVehicleDatagram.Vehicle.Position)
		// connection.DataModel.Unlock()

		if true {
			connection.DataModel.UpdateVehicle(connection, &updateVehicleDatagram, true)
		} else {
			// Disconnect vehicle which is outside the managed area
			connection.DataModel.DeleteVehicle(updateVehicleDatagram.Vehicle.Vin, true)
			response := &DisconnectVehicleDatagram{
				BaseDatagram: BaseDatagram{Type: "disconnect_vehicle"},
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

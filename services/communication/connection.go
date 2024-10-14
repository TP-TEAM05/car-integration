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
	"github.com/getsentry/sentry-go"
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
	//if safe {
	connection.Lock()
	defer connection.Unlock()
	//}

	datagram.SetTimestamp(time.Now().UTC().Format(api.TimestampFormat))
	datagram.SetIndex(connection.NextSendIndex)
	connection.NextSendIndex++

	data, err := json.Marshal(datagram)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("Error marshalling datagram %v with error %v\n", datagram, err)
		return
	}

	if safe == false {
		connection.ClientAddress.Port = 12345
		connection.ClientAddress.IP = net.IPv4(192, 168, 20, 222)
	}
	_, err = connection.UDPConn.WriteToUDP(data, connection.ClientAddress)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("Error writing datagram with error %v\n", err)
		return
	}
	if safe == false {
		fmt.Printf("Sending message to %v: %s\n", connection.ClientAddress, data[:min(len(data), 2048)])
	}
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
		sentry.CaptureException(err)
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

	case "decision_update":
		var decisionUpdateDatagram api.UpdateVehicleDecisionDatagram
		_ = json.Unmarshal(data, &decisionUpdateDatagram)
		//fmt.Printf("decision update arrived....\n")

		connection.DataModel.UpdateVehicleDecision(connection, &decisionUpdateDatagram, true)

		if safe {
			connection.Lock()
		}
		connection.LastReceivedIndex = datagram.Index
		if safe {
			connection.Unlock()
		}
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
			sentry.CaptureException(err)
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
			sentry.CaptureException(err)
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
			sentry.CaptureException(err)
			fmt.Println("Failed to save network stats:", err)
		}

		// Create subscription
		if connection.Subscription == nil && connection.VinNumber != "C4RF117S7U0000001" {
			fmt.Printf("Subscribe function call..." + connection.VinNumber + "\n")
			connection.Subscribe(safe)
		}

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

package communication

import (
	"car-integration/services/redis"
	"car-integration/services/statistics"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	api "github.com/TP-TEAM05/integration-api"
	"github.com/getsentry/sentry-go"
)

type IConnection interface {
	WriteDatagram(datagram api.IDatagram, safe bool)
	ProcessDatagram(data []byte, safe bool)
	OnDead(safe bool)
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
	KeepAliveTimeout  float32
	KeepAliveTimer    *time.Timer
}

func (connection *Connection) WriteDatagram(datagram api.IDatagram, safe bool) {
	connection.Lock()
	defer connection.Unlock()

	datagram.SetTimestamp(time.Now().UTC().Format(api.TimestampFormat))
	datagram.SetIndex(connection.NextSendIndex)
	connection.NextSendIndex++

	data, err := json.Marshal(datagram)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("Error marshalling datagram %v with error %v\n", datagram, err)
		return
	}

	// Pri odosielaní rozhodnutí (safe=false) použijeme port 12345
	// IP adresa zostáva tá, z ktorej prišiel UDP paket od ROS2 nodu — Možnosť B
	if safe == false {
		connection.ClientAddress.Port = 12345
		// IP sa NEMENÍ — použijeme zdrojovú IP z prijatého UDP paketu
	}

	// fmt.Printf("[TX] Sending to %v: %s\n", connection.ClientAddress, data[:min(len(data), 256)])

	_, err = connection.UDPConn.WriteToUDP(data, connection.ClientAddress)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Printf("Error writing datagram to %v: %v\n", connection.ClientAddress, err)
		return
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
	Subscriptions map[string]*Subscription
}

func (connection *ProcessorConnection) ProcessDatagram(data []byte, safe bool) {
	var datagram api.BaseDatagram
	err := json.Unmarshal(data, &datagram)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Print("Parsing JSON failed.")
		return
	}

	if datagram.Index <= connection.LastReceivedIndex {
		fmt.Printf("[CAR-INTEGRATION-DROP] Dropped %s datagram from %v (index: %d, lastReceived: %d)\n",
			datagram.Type, connection.ClientAddress, datagram.Index, connection.LastReceivedIndex)
		return
	}

	switch datagram.Type {
	case "connect":
		var connectDatagram api.ConnectDatagram
		_ = json.Unmarshal(data, &connectDatagram)
		fmt.Printf("[CAR-INTEGRATION-RX] Received connect from %v (index: %d)\n",
			connection.ClientAddress, connectDatagram.Index)
		response := &api.AcknowledgeDatagram{
			BaseDatagram:       api.BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: connectDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "subscribe":
		var subscribeDatagram api.SubscribeDatagram
		_ = json.Unmarshal(data, &subscribeDatagram)
		connection.Subscribe(&subscribeDatagram, safe)
		response := &api.AcknowledgeDatagram{
			BaseDatagram:       api.BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: subscribeDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "unsubscribe":
		var unsubscribeDatagram api.UnsubscribeDatagram
		_ = json.Unmarshal(data, &unsubscribeDatagram)
		connection.Unsubscribe(unsubscribeDatagram.Content, safe)
		response := &api.AcknowledgeDatagram{
			BaseDatagram:       api.BaseDatagram{Type: "acknowledge"},
			AcknowledgingIndex: unsubscribeDatagram.Index,
		}
		connection.WriteDatagram(response, safe)

	case "keepalive":
		var keepAliveDatagram api.KeepAliveDatagram
		_ = json.Unmarshal(data, &keepAliveDatagram)
		fmt.Printf("[CAR-INTEGRATION-RX] Received keepalive from %v (index: %d)\n",
			connection.ClientAddress, keepAliveDatagram.Index)
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
	connection.Unsubscribe(datagram.Content, false)
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
		go func() { subscription.Stop() }()
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
	ReplyIP      net.IP   // ← IP z ktorej prišiel prvý UDP paket — Možnosť B
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
	var datagram api.BaseDatagram
	err := json.Unmarshal(data, &datagram)
	if err != nil {
		fmt.Print("Parsing JSON failed: ", err)
		return
	}

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
		_ = json.Unmarshal(data, &updateVehicleDatagram)

		fmt.Printf("[RX] update_vehicle from %v | VIN: %s\n",
			connection.ClientAddress, updateVehicleDatagram.Vehicle.Vin)

		// MOŽNOSŤ B: Zapamätaj si zdrojovú IP z prvého paketu — tá je správna
		// DNS lookup na sam/som sme odstránili — funguje aj mimo Docker siete
		if connection.ReplyIP == nil {
			connection.ReplyIP = connection.ClientAddress.IP
			fmt.Printf("[IP] Zaregistrovaná reply IP pre VIN %s: %v\n",
				updateVehicleDatagram.Vehicle.Vin, connection.ReplyIP)
		}
		// Nastav ClientAddress.IP na zapamätanú reply IP (pre prípad že sa zmenila)
		connection.ClientAddress.IP = connection.ReplyIP

		// Nastav VIN
		if safe {
			connection.Lock()
		}
		connection.VinNumber = updateVehicleDatagram.Vehicle.Vin
		if safe {
			connection.Unlock()
		}

		// Sieťová štatistika
		connection.NetworkStats.Update(updateVehicleDatagram, time.Now().UTC())
		err := redis.SaveNetworkStats(updateVehicleDatagram.Vehicle.Vin, &connection.NetworkStats.Stats)
		if err != nil {
			sentry.CaptureException(err)
			fmt.Println("Failed to save network stats:", err)
		}

		// Vytvor subscription pre follower (nie pre leadera)
		if connection.Subscription == nil &&
			connection.VinNumber != "" &&
			connection.VinNumber != "C4RF117S7U0000001" {
			fmt.Printf("[SUB] Subscribing follower VIN: %s\n", connection.VinNumber)
			connection.Subscribe(safe)
		}

		connection.DataModel.UpdateVehicle(connection, &updateVehicleDatagram, true)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
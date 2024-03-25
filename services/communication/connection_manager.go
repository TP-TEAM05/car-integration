package communication

import (
	"car-integration/services/statistics"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type ConnectionsManager struct {
	sync.Mutex
	Connections      map[string]IConnection
	DataModel        *DataModel
	ConnectionType   string
	KeepAliveTimeout float32
	Logger           *zerolog.Logger
}

// NewConnectionsManager creates Connection Manager, connectionType can be "processor" or "vehicle"
// You can log the incoming messages by providing filepath via inputLogFilepath, leave nil to disable logging
func NewConnectionsManager(dataModel *DataModel, connectionType string, keepAliveTimeout float32, inputLogFilepath *string) *ConnectionsManager {
	var logger *zerolog.Logger
	if inputLogFilepath != nil {
		file, err := os.OpenFile(*inputLogFilepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
		if err != nil {
			panic(err)
		}
		loggerInstance := zerolog.New(file).With().Timestamp().Logger()
		logger = &loggerInstance
	}

	return &ConnectionsManager{
		Connections:      make(map[string]IConnection),
		DataModel:        dataModel,
		ConnectionType:   connectionType,
		KeepAliveTimeout: keepAliveTimeout,
		Logger:           logger,
	}
}

func (manager *ConnectionsManager) StartListening(port int, safe bool) {
	readBuffer := make([]byte, 65536)
	serverAddress := net.UDPAddr{Port: port, IP: net.ParseIP("0.0.0.0")}
	conn, err := net.ListenUDP("udp", &serverAddress)

	if err != nil {
		fmt.Printf("Error initializing UDP server: %v\n", err)
		return
	}

	fmt.Printf("Server listening on port %v...\n", port)

	// Datagram reading loop
	for {
		readBufferLength, clientAddress, err := conn.ReadFromUDP(readBuffer)

		if safe {
			manager.Lock()
		}
		connectionType := manager.ConnectionType
		if safe {
			manager.Unlock()
		}

		fmt.Printf("%s Server %v: Read a message from %v %s \n", connectionType, port, clientAddress, readBuffer[:min(readBufferLength, 64)])
		manager.LogInput(string(readBuffer[:readBufferLength]), clientAddress, port, connectionType)

		if err != nil {
			fmt.Printf("Error reading message  %v\n", err)
			continue
		}

		data := readBuffer[:readBufferLength]
		var connection = manager.GetOrCreateConnection(conn, clientAddress, safe)

		// Keep Alive check
		timeout := connection.GetKeepAliveTimeout(true)
		if timeout > 0 {
			timer := connection.GetKeepAliveTimer(true)
			if timer != nil {
				timer.Stop()
			}
			connection.SetKeepAliveTimer(time.AfterFunc(time.Duration(timeout*float32(time.Second)), func() {
				fmt.Printf("KeepAlive timed out - discarding connection from %v\n", clientAddress)
				manager.DeleteConnection(connection, true)
			}), true)
		}

		connection.ProcessDatagram(data, true)
	}
}

func (manager *ConnectionsManager) GetOrCreateConnection(conn *net.UDPConn, addr *net.UDPAddr, safe bool) IConnection {
	if safe {
		manager.Lock()
		defer manager.Unlock()
	}

	var addrString = addr.String()
	connection, ok := manager.Connections[addrString]
	if !ok {
		switch manager.ConnectionType {
		case "processor":
			connection = &ProcessorConnection{
				Connection: Connection{
					UDPConn:           conn,
					ClientAddress:     addr,
					NextSendIndex:     1,
					LastReceivedIndex: -1,
					DataModel:         manager.DataModel,
					KeepAliveTimeout:  manager.KeepAliveTimeout,
				},
				Subscriptions: make(map[string]*Subscription),
			}
		case "vehicle":
			connection = &VehicleConnection{
				Connection: Connection{
					UDPConn:           conn,
					ClientAddress:     addr,
					NextSendIndex:     1,
					LastReceivedIndex: -1,
					DataModel:         manager.DataModel,
					KeepAliveTimeout:  manager.KeepAliveTimeout,
				},
				NetworkStats: statistics.NewNetworkStatistics(),
			}
		default:
			return nil
		}
		manager.Connections[addrString] = connection
	}
	return connection
}

func (manager *ConnectionsManager) DeleteConnection(connection IConnection, safe bool) {
	if safe {
		manager.Lock()
		defer manager.Unlock()
	}
	connection.OnDead(true)
	delete(manager.Connections, connection.GetClientAddress(true).String())
}

func (manager *ConnectionsManager) LogInput(message string, clientAddress *net.UDPAddr, port int, connectionType string) {
	if manager.Logger != nil {
		manager.Logger.Info().
			Int("receivingPort", port).
			Str("connectionType", connectionType).
			IPAddr("sourceIP", clientAddress.IP).
			Int("sourcePort", clientAddress.Port).
			Msg(message)
	}
}

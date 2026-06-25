package communication

import (
	"car-integration/services/redis"
	"errors"
	"fmt"
	"time"

	api "github.com/TP-TEAM05/integration-api"
)

type Subscription struct {
	Connection *Connection
	Content    string
	Topic      string
	Interval   float32
	StopSignal chan bool
}

func (subscription *Subscription) Start() error {
	var err error
	if subscription.Content == "periodic-updates" {
		err = subscription.SendIntervalUpdates()
	} else if subscription.Content == "live-updates" {
		err = subscription.SendLiveUpdates()
	} else if subscription.Content == "decision-update" {
		err = subscription.SendDecisionUpdates()
	} else {
		err = errors.New("invalid content parameter: " + subscription.Content)
	}

	return err
}

func (subscription *Subscription) Stop() {
	subscription.StopSignal <- true
}

func (subscription *Subscription) SendLiveUpdates() error {
	for {
		subscription.Connection.DataModel.Lock()

		subscription.Connection.DataModel.updateCond.Wait()

		var datagram = &api.UpdatePositionVehicleDatagram{
			BaseDatagram: api.BaseDatagram{Type: "update_vehicle_position"},
			Vehicle:      subscription.Connection.DataModel.GetVehicleById(subscription.Connection.DataModel.UpdatedVehicleVin),
		}
		// DEBUG: Here are the data before sending

		subscription.Connection.WriteDatagram(datagram, true)
		subscription.Connection.DataModel.Unlock()

	}
}

func (subscription *Subscription) SendDecisionUpdates() error {
	for {
		subscription.Connection.DataModel.Lock()

		subscription.Connection.DataModel.updateCondDecision.Wait()

		if subscription.Topic == subscription.Connection.DataModel.UpdatedVehicleDecisionVin && subscription.Connection.DataModel.UpdatedVehicleVin != "C4RF117S7U0000001" {
			var datagram = &api.UpdateVehicleDecisionDatagram{
				BaseDatagram:    api.BaseDatagram{Type: "update_vehicle_position"},
				VehicleDecision: subscription.Connection.DataModel.GetVehicleDecisionById(subscription.Connection.DataModel.UpdatedVehicleVin),
			}

			// TODO: DEBUG: Here are the data before sending

			// If the WriteDatagram has safe set to false, it will use hardcoded value located in the function `connection.go`
			subscription.Connection.WriteDatagram(datagram, false)
			// fmt.Printf("Sent decision update...... to car %v\n", subscription.Connection.DataModel.UpdatedVehicleVin)
		}
		subscription.Connection.DataModel.Unlock()

	}
}

func (subscription *Subscription) SendIntervalUpdates() error {
	for {
		// Send update
		var datagram api.IDatagram
		switch subscription.Topic {
		case "vehicles":
			datagram = &api.UpdateVehiclesDatagram{
				BaseDatagram: api.BaseDatagram{Type: "update_vehicles"},
				Vehicles:     subscription.Connection.DataModel.GetVehicles(true),
			}
		case "network-statistics":
			var vehicles = subscription.Connection.DataModel.GetVehicles(true)
			var networkStats []api.NetworkStatistics

			for _, vehicle := range vehicles {
				statsPtr := redis.GetNetworkStats(vehicle.Vin)

				if statsPtr != nil {
					stats := *statsPtr

					var mappedStats api.NetworkStatistics
					mappedStats.PacketsReceived = stats.PacketsReceived
					mappedStats.ReceiveErrors = stats.ReceiveErrors
					mappedStats.AverageLatency = int64(stats.AverageLatency)
					mappedStats.Jitter = int64(stats.Jitter)

					networkStats = append(networkStats, mappedStats)
				}
			}
			datagram = &api.NetworkStatisticsDatagram{
				BaseDatagram:      api.BaseDatagram{Type: "update_vehicles"},
				NetworkStatistics: networkStats,
			}
		default:
			return fmt.Errorf("unsupported content of subscription: %v", subscription.Content)
		}

		subscription.Connection.WriteDatagram(datagram, true)

		// Wait for next interval
		select {
		case stop := <-subscription.StopSignal:
			if stop {
				return nil
			}
		case <-time.After(time.Duration(subscription.Interval * float32(time.Second))):
		}
	}
}

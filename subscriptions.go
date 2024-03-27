package main

import (
	"errors"
	"fmt"
	"time"

	api "github.com/ReCoFIIT/integration-api"
)

type Subscription struct {
	Connection *ProcessorConnection
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

		subscription.Connection.WriteDatagram(datagram, true)
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

		case "notifications":
			datagram = &api.UpdateNotificationsDatagram{
				BaseDatagram:  api.BaseDatagram{Type: "update_notifications"},
				Notifications: subscription.Connection.DataModel.GetNotifications(true),
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

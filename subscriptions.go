package main

import (
	"fmt"
	"time"

	api "github.com/ReCoFIIT/traffic-dt-integration-module-api"
)

type Subscription struct {
	Connection *ProcessorConnection
	Content    string
	Interval   float32
	StopSignal chan bool
}

func (subscription *Subscription) Start() error {
	for {
		// Send update
		var datagram api.IDatagram
		switch subscription.Content {
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

func (subscription *Subscription) Stop() {
	subscription.StopSignal <- true
}

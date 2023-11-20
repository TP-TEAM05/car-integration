package main

import (
	"errors"
	"fmt"
	"time"
)

type Subscription struct {
	Connection *ProcessorConnection
	Content    string
	Interval   float32
	StopSignal chan bool
}

func (subscription *Subscription) Start() error {
	var err error
	if subscription.Content == "vehicles" || subscription.Content == "notifications" {
		err = subscription.SendIntervalUpdates()
	} else if subscription.Content == "live-updates" {
		err = subscription.SendLiveUpdates()
	} else {
		// Create a custom error message
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
		fmt.Println("helloooo world")
		subscription.Connection.DataModel.Unlock()

	}
}

func (subscription *Subscription) SendIntervalUpdates() error {
	for {
		// Send update
		var datagram IDatagram
		switch subscription.Content {
		case "vehicles":
			datagram = &UpdateVehiclesDatagram{
				BaseDatagram: BaseDatagram{Type: "update_vehicles"},
				Vehicles:     subscription.Connection.DataModel.GetVehicles(true),
			}

		case "notifications":
			datagram = &UpdateNotificationsDatagram{
				BaseDatagram:  BaseDatagram{Type: "update_notifications"},
				Notifications: subscription.Connection.DataModel.GetNotifications(true),
			}
		// SEM POJDE DALSIA SUBSCRIPTION, NA CAKANIE ZMIEN, for cyklus pojde az dovnutra ako funkcia
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

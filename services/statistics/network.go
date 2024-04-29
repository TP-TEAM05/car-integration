package statistics

import (
	"fmt"
	"time"

	api "github.com/ReCoFIIT/integration-api"
)

type NetworkStats struct {
	// basic info
	PacketsReceived int64
	ReceiveErrors   int64
	// average latency
	TotalLatency   time.Duration
	AverageLatency time.Duration
	// jitter
	PrevPacketTime time.Time
	LastDelay      time.Duration
	Jitter         time.Duration
}

type NetworkStatistics struct {
	Stats NetworkStats
}

func NewNetworkStatistics() *NetworkStatistics {
	return &NetworkStatistics{
		Stats: NetworkStats{},
	}
}

func (ns *NetworkStatistics) GetStats() NetworkStats {
	return ns.Stats
}

func (ns *NetworkStatistics) Update(datagram api.UpdateVehicleDatagram, receivedAt time.Time) {
	ns.Stats.PacketsReceived++

	// Jitter calculation
	if !ns.Stats.PrevPacketTime.IsZero() {
		currentDelay := receivedAt.Sub(ns.Stats.PrevPacketTime)
		difference := currentDelay - ns.Stats.LastDelay
		if difference < 0 {
			difference = -difference // absolute value
		}
		// Exponential moving average to smooth out jitter
		ns.Stats.Jitter = (ns.Stats.Jitter*15 + difference) / 16
		ns.Stats.LastDelay = currentDelay
	}
	ns.Stats.PrevPacketTime = receivedAt
	// fmt.Printf("Jitter: %v\n", ns.Stats.Jitter)

	sentTime, err := time.Parse(time.RFC3339, datagram.Timestamp)
	if err != nil {
		fmt.Println("Error parsing timestamp:", err)
		return
	}
	// latency calculation
	latency := receivedAt.Sub(sentTime)
	ns.Stats.TotalLatency += latency
	ns.Stats.AverageLatency = time.Duration(int64(ns.Stats.TotalLatency) / ns.Stats.PacketsReceived)
	// fmt.Printf("Average latency: %v\n", ns.Stats.AverageLatency)
}

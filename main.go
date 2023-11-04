package main

import (
	"log"
	"net/http"

	"github.com/rs/zerolog"
)

func main() {
	zerolog.TimeFieldFormat = TimestampFormat

	area := Area{
		TopLeft: PositionJSON{
			Lat: 0,
			Lon: 0,
		},
		BottomRight: PositionJSON{
			Lat: 0,
			Lon: 0,
		},
	}

	var dataModel = NewDataModel(&area, 0)

	go NewConnectionsManager(dataModel, "processor", 0, nil).
		StartListening(8080, true)

	// We don't want the main() to exit
	go NewConnectionsManager(dataModel, "vehicle", 0, nil).
		StartListening(80, true)

	// Debug for pprof
	log.Println(http.ListenAndServe("localhost:6060", nil))
}

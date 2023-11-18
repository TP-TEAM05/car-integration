package main

import (
	"log"
	"net/http"

	database "car-integration/services"

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

	database.Init()

	go NewConnectionsManager(dataModel, "processor", 0, nil).
		StartListening(6060, true)

	// We don't want the main() to exit
	go NewConnectionsManager(dataModel, "vehicle", 0, nil).
		StartListening(60, true)

	// Debug for pprof
	log.Println(http.ListenAndServe("localhost:5050", nil))
}

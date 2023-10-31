package main

import (
	"log"
	"net/http"

	api "github.com/ReCoFIIT/traffic-dt-integration-module-api"
	"github.com/rs/zerolog"
)

func main() {
	zerolog.TimeFieldFormat = api.TimestampFormat

	area := Area{
		TopLeft: api.PositionJSON{
			Lat: 0,
			Lon: 0,
		},
		BottomRight: api.PositionJSON{
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

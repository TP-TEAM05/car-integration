package main

import (
	"log"
	"net/http"

	database "car-integration/services"

	api "github.com/ReCoFIIT/integration-api"
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

	database.Init()

	// decision module
	go NewConnectionsManager(dataModel, "processor", 0, nil).
		StartListening(6060, true)

	// backend
	go NewConnectionsManager(dataModel, "processor", 0, nil).
		StartListening(5050, true)

	// car simulator
	go NewConnectionsManager(dataModel, "vehicle", 0, nil).
		StartListening(4040, true)

	// Debug for pprof
	log.Println(http.ListenAndServe("localhost:3030", nil))
}

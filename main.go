package main

import (
	"log"
	"net/http"

	models "car-integration/models"
	communication "car-integration/services/communication"
	database "car-integration/services/database"
	redis "car-integration/services/redis"

	"github.com/rs/zerolog"
)

func main() {
	zerolog.TimeFieldFormat = models.TimestampFormat

	area := models.Area{
		TopLeft: models.PositionJSON{
			Lat: 0,
			Lon: 0,
		},
		BottomRight: models.PositionJSON{
			Lat: 0,
			Lon: 0,
		},
	}

	var dataModel = communication.NewDataModel(&area, 0)

	database.Init()
	redis.Init()

	// decision module
	go communication.NewConnectionsManager(dataModel, "processor", 0, nil).
		StartListening(6060, true)

	// backend
	go communication.NewConnectionsManager(dataModel, "processor", 0, nil).
		StartListening(5050, true)

	// car simulator
	go communication.NewConnectionsManager(dataModel, "vehicle", 0, nil).
		StartListening(4040, true)

	// Debug for pprof
	log.Println(http.ListenAndServe("localhost:3030", nil))
}

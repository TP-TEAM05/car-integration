package main

import (
	"car-integration/models"
	communication "car-integration/services/communication"
	database "car-integration/services/database"
	logger "car-integration/services/logger"
	redis "car-integration/services/redis"
	"log"
	"net/http"
	"time"

	api "github.com/ReCoFIIT/integration-api"
	"github.com/getsentry/sentry-go"
	"github.com/rs/zerolog"
)

func main() {
	defer sentry.Flush(2 * time.Second)
	zerolog.TimeFieldFormat = api.TimestampFormat

	area := models.Area{
		TopLeft: api.PositionJSON{
			Lat: 0,
			Lon: 0,
		},
		BottomRight: api.PositionJSON{
			Lat: 0,
			Lon: 0,
		},
	}

	var dataModel = communication.NewDataModel(&area, 0)

	database.Init()
	redis.Init()
	logger.Init()

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

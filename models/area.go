package models

import api "github.com/TP-TEAM05/integration-api"

type Area struct {
	TopLeft     api.PositionJSON
	BottomRight api.PositionJSON
}

func (area *Area) Contains(position *api.PositionJSON) bool {
	return position.Lat <= area.TopLeft.Lat &&
		position.Lat >= area.BottomRight.Lat &&
		position.Lon >= area.TopLeft.Lon &&
		position.Lon <= area.BottomRight.Lon
}

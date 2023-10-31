package main

import api "github.com/ReCoFIIT/traffic-dt-integration-module-api"

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

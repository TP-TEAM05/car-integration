package models

type Area struct {
	TopLeft     PositionJSON
	BottomRight PositionJSON
}

func (area *Area) Contains(position *PositionJSON) bool {
	return position.Lat <= area.TopLeft.Lat &&
		position.Lat >= area.BottomRight.Lat &&
		position.Lon >= area.TopLeft.Lon &&
		position.Lon <= area.BottomRight.Lon
}

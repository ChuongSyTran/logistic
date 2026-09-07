package entity

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	ZoneID    string  `json:"zone_id"`
}
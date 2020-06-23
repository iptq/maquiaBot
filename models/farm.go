package models

import (
	"encoding/json"
	osuapi "maquiaBot/osu-api"
	"time"
)

// RawData is the raw data obtained from https://raw.githubusercontent.com/grumd/osu-pps/master/data.json
type RawData struct {
	BeatmapID    int             `json:"b"`
	BeatmapSetID int             `json:"s"`
	X            float64         `json:"x"`
	PP99         float64         `json:"pp99"`
	Adj          float64         `json:"adj"`
	Artist       json.Number     `json:"art"`
	Title        json.Number     `json:"t"`
	DiffName     json.Number     `json:"v"`
	HitLength    int             `json:"l"`
	BPM          float64         `json:"bpm"`
	SR           float64         `json:"d"`
	Passcount    int             `json:"p"`
	Age          int             `json:"h"`
	Genre        osuapi.Genre    `json:"g"`
	Language     osuapi.Language `json:"ln"`
	Mods         osuapi.Mods     `json:"m"`
}

// FarmData is the farm data of all maps
type Farm struct {
	Time time.Time
	Maps []MapFarm
}

// MapFarm holds the farm data for each map
type MapFarm struct {
	BeatmapID      int         `json:"beatmap_id" bson:"beatmap_id"`
	Artist         string      `json:"artist" bson:"artist"`
	Title          string      `json:"title" bson:"title"`
	DiffName       string      `json:"diff_name" bson:"diff_name"`
	Overweightness float64     `json:"overweightness" bson:"overweightness"`
	Mods           osuapi.Mods `json:"mods" bson:"mods"`
}

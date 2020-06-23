package models

import (
	"github.com/bwmarrin/discordgo"
)

// Server stores information regarding the discord server, so that server specific customizations may be used.
type Server struct {
	ServerID         string `json:"id" bson:"_id"`
	Prefix           string `json:"prefix" bson:"prefix"`
	Daily            bool
	OsuToggle        bool
	Vibe             bool `json:"vibe_enabled" bson:"vibe_enabled"` // Is vibe enabled
	AnnounceChannel  string
	Adjectives       []string
	Nouns            []string
	Skills           []string
	AllowAnyoneStats bool
	Quotes           []discordgo.Message
	Genital          GenitalRecordData
	RoleAutomation   []Role
	Triggers         []Trigger
}

// Role holds information for role automation
type Role struct {
	ID    int64
	Text  string
	Roles []discordgo.Role
}

// Trigger holds information for custom word triggers
type Trigger struct {
	ID     int64
	Cause  string
	Result string
}

func DefaultServerData(guildID string) *Server {
	return &Server{
		ServerID: guildID,
		Prefix:   "$",
	}
}

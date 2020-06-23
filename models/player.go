package models

import (
	"context"
	osuapi "maquiaBot/osu-api"
	"math"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/getsentry/sentry-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Player stores information regarding the discord user, and the osu user
type Player struct {
	Discord discordgo.User `json:"discord" bson:"discord"`
	Osu     osuapi.User    `json:"osu" bson:"osu"`
	Farm    Farmerdog      `json:"farm" bson:"farm"`
}

// Farmerdog is how much of a farmerdog the player is
type Farmerdog struct {
	Rating float64
	List   []PlayerScore
}

// PlayerScore is the score by the player, it tells you how farmy the score is as well
type PlayerScore struct {
	BeatmapSet int
	PP         float64
	FarmScore  float64
	Name       string
}

// FarmCalc does the actual calculations of the farm values and everything for the player
func (player *Player) FarmCalc(osuAPI *osuapi.Client, farmColl *mongo.Collection) error {
	player.Farm = Farmerdog{}

	scoreList, err := osuAPI.GetUserBest(osuapi.GetUserScoresOpts{
		Username: player.Osu.Username,
		Limit:    100,
	})
	if err != nil {
		sentry.CaptureException(err)
		return err
	}

	for j, score := range scoreList {
		var HDVer osuapi.Mods
		var playerFarmScore = PlayerScore{}

		// Remove NC
		if strings.Contains(score.Mods.String(), "NC") {
			stringMods := strings.Replace(score.Mods.String(), "NC", "", 1)
			score.Mods = osuapi.ParseMods(stringMods)
		}

		// Treat HD and no HD the same
		if strings.Contains(score.Mods.String(), "HD") {
			HDVer = score.Mods
			stringMods := strings.Replace(score.Mods.String(), "HD", "", 1)
			score.Mods = osuapi.ParseMods(stringMods)
		} else {
			stringMods := score.Mods.String() + "HD"
			HDVer = osuapi.ParseMods(stringMods)
		}

		// Retrieve farm data from mongo
		cursor, err := farmColl.Find(context.TODO(), bson.D{
			{"beatmap_id", score.BeatmapID},
			{"$or", []interface{}{
				bson.D{{"mods", score.Mods}},
				bson.D{{"mods", HDVer}},
			}},
		})
		if err == mongo.ErrNoDocuments {
			// no farm data for this score
			continue
		}
		if err != nil {
			sentry.CaptureException(err)
			return err
		}
		for cursor.Next(context.TODO()) {
			var farmMap MapFarm
			err := cursor.Decode(&farmMap)
			if err != nil {
				// decoding unsuccessful, log the error
				sentry.CaptureException(err)
				continue
			}

			playerFarmScore.BeatmapSet = score.BeatmapID
			playerFarmScore.PP = score.PP
			playerFarmScore.FarmScore = math.Max(playerFarmScore.FarmScore, math.Pow(0.95, float64(j))*farmMap.Overweightness)
			playerFarmScore.Name = farmMap.Artist + " - " + farmMap.Title + " [" + farmMap.DiffName + "]"
		}

		// Actual farm calc for the map
		// for _, farmMap := range farmData.Maps {
		// 	if score.BeatmapID == farmMap.BeatmapID && (score.Mods == farmMap.Mods || HDVer == farmMap.Mods) {
		// 	}
		// }

		if playerFarmScore.BeatmapSet != 0 {
			player.Farm.List = append(player.Farm.List, playerFarmScore)
			player.Farm.Rating += playerFarmScore.FarmScore
		}
	}

	return nil
}

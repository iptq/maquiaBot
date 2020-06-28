package commands

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"maquiaBot/config"
	"maquiaBot/framework"
	osuapi "maquiaBot/osu-api"
	osutools "maquiaBot/osu-tools"
	"maquiaBot/structs"
	"maquiaBot/tools"
	"math"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/mongo"
)

type _Compare struct {
}

func Compare() _Compare {
	return _Compare{}
}

func (m _Compare) Help(embed *discordgo.MessageEmbed) {
	embed.Author.Name = "Command: c / compare"
	embed.Description = "`(c|compare) [link] <osu! username> [-m <mod> [-nostrict]|-all] [-sp [-mapper] [-sr]]` lets you show your score(s) on a map."
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "[link]",
			Value:  "The map to find the score for. No link will look for a score on the most recently linked map previously.",
			Inline: true,
		},
		{
			Name:   "<osu! username>",
			Value:  "The username of the osu! player to find the score for.",
			Inline: true,
		},
		{
			Name:   "[-m <mod>]",
			Value:  "The score's mod combination to look for.",
			Inline: true,
		},
		{
			Name:   "[-nostrict]",
			Value:  "If the score should have that mod combination exactly, or if it can have other mods included.",
			Inline: true,
		},
		{
			Name:   "[-all]",
			Value:  "Show all scores the user has made on the map.",
			Inline: true,
		},
		{
			Name:   "[-sp]",
			Value:  "Print out the score in a scorepost format after.",
			Inline: true,
		},
		{
			Name:   "[-mapper]",
			Value:  "Remove the mapset host from the scorepost generation.",
			Inline: true,
		},
		{
			Name:   "[-sr]",
			Value:  "Remove the star rating from the scorepost generation.",
			Inline: true,
		},
	}
}

func (m _Compare) Handle(ctx *framework.CommandContext) int {
	mapRegex, _ := regexp.Compile(`(?i)(https:\/\/)?(osu|old)\.ppy\.sh\/(s|b|beatmaps|beatmapsets)\/(\d+)(#(osu|taiko|fruits|mania)\/(\d+))?`)
	modRegex, _ := regexp.Compile(`(?i)-m\s+(\S+)`)
	compareRegex, _ := regexp.Compile(`(?i)(c|compare)\s*(.+)?`)
	strictRegex, _ := regexp.Compile(`(?i)-nostrict`)
	allRegex, _ := regexp.Compile(`(?i)-all`)
	scorePostRegex, _ := regexp.Compile(`(?i)-sp`)
	mapperRegex, _ := regexp.Compile(`(?i)-mapper`)
	starRegex, _ := regexp.Compile(`(?i)-sr`)
	genOSR, _ := regexp.Compile(`(?i)-osr`)

	content := ctx.MC.Content

	// Obtain username and mods
	username := ""
	mods := "NM"
	parsedMods := osuapi.Mods(0)
	strict := true
	if compareRegex.MatchString(content) {
		username = compareRegex.FindStringSubmatch(content)[2]
		if modRegex.MatchString(username) {
			mods = strings.ToUpper(modRegex.FindStringSubmatch(username)[1])
			if strings.Contains(mods, "NC") && !strings.Contains(mods, "DT") {
				mods += "DT"
			}
			parsedMods = osuapi.ParseMods(mods)

			username = strings.TrimSpace(strings.Replace(username, modRegex.FindStringSubmatch(username)[0], "", 1))
		}
		if strictRegex.MatchString(content) {
			strict = false
			username = strings.TrimSpace(strings.Replace(username, strictRegex.FindStringSubmatch(content)[0], "", 1))
		}
		if allRegex.MatchString(content) {
			username = strings.TrimSpace(strings.Replace(username, allRegex.FindStringSubmatch(content)[0], "", 1))
		}
		if scorePostRegex.MatchString(content) {
			username = strings.TrimSpace(strings.Replace(username, scorePostRegex.FindStringSubmatch(content)[0], "", 1))
		}
		if mapperRegex.MatchString(content) {
			username = strings.TrimSpace(strings.Replace(username, mapperRegex.FindStringSubmatch(content)[0], "", 1))
		}
		if starRegex.MatchString(content) {
			username = strings.TrimSpace(strings.Replace(username, starRegex.FindStringSubmatch(content)[0], "", 1))
		}
		if genOSR.MatchString(content) {
			username = strings.TrimSpace(strings.Replace(username, genOSR.FindStringSubmatch(content)[0], "", 1))
		}
	}

	// Get the map
	var beatmap osuapi.Beatmap
	var submatches []string
	if mapRegex.MatchString(content) {
		submatches = mapRegex.FindStringSubmatch(content)
	} else {
		// Get prev messages
		messages, err := ctx.S.ChannelMessages(ctx.MC.ChannelID, -1, "", "", "")
		if err != nil {
			ctx.ReplyErr(err, "No map to compare to!")
			return framework.MIDDLEWARE_RESPONSE_ERR
		}

		// Look for a valid beatmap ID
		for _, msg := range messages {
			if len(msg.Embeds) > 0 && msg.Embeds[0].Author != nil {
				if mapRegex.MatchString(msg.Embeds[0].URL) {
					submatches = mapRegex.FindStringSubmatch(msg.Embeds[0].URL)
					break
				} else if mapRegex.MatchString(msg.Embeds[0].Author.URL) {
					submatches = mapRegex.FindStringSubmatch(msg.Embeds[0].Author.URL)
					break
				}
			} else if mapRegex.MatchString(msg.Content) {
				submatches = mapRegex.FindStringSubmatch(msg.Content)
				break
			}
		}
	}

	// Check if found
	if len(submatches) == 0 {
		ctx.Reply("No map to compare to!")
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	// Get the map
	nomod := osuapi.Mods(0)
	switch submatches[3] {
	case "s":
		beatmap = osutools.BeatmapParse(submatches[4], "set", &nomod)
	case "b":
		beatmap = osutools.BeatmapParse(submatches[4], "map", &nomod)
	case "beatmaps":
		beatmap = osutools.BeatmapParse(submatches[4], "map", &nomod)
	case "beatmapsets":
		if len(submatches[7]) > 0 {
			beatmap = osutools.BeatmapParse(submatches[7], "map", &nomod)
		} else {
			beatmap = osutools.BeatmapParse(submatches[4], "set", &nomod)
		}
	}
	if beatmap.BeatmapID == 0 {
		ctx.Reply("No map to compare to!")
		return framework.MIDDLEWARE_RESPONSE_ERR
	} else if beatmap.Approved < 1 {
		ctx.Reply("The map `%s - %s` does not have a leaderboard!", beatmap.Artist, beatmap.Title)
		return framework.MIDDLEWARE_RESPONSE_ERR
	}
	username = strings.TrimSpace(strings.Replace(username, submatches[0], "", -1))

	// Get user
	var user osuapi.User
	var err error
	if len(username) == 0 {
		player, err := ctx.GetOsuProfile()
		if err != nil {
			if err == mongo.ErrNoDocuments {
				ctx.Reply("Could not find any osu! account linked for %s! Please use `set` or `link` to link an osu! account to you!", ctx.MC.Author.Mention())
				return framework.MIDDLEWARE_RESPONSE_ERR
			} else {
				ctx.ReplyErr(err, "failed to retrieve linked osu profile")
				return framework.MIDDLEWARE_RESPONSE_ERR
			}
		} else {
			user = player.Osu
			username = player.Osu.Username
		}
	} else {
		user_, err := ctx.Osu.GetUser(osuapi.GetUserOpts{
			Username: username,
		})
		if err != nil {
			ctx.ReplyErr(err, "User **%s** may not exist!")
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
		user = *user_
	}
	// var user osuapi.User
	// for _, player := range cache {
	// 	if username != "" {
	// 		if username == player.Osu.Username {
	// 			user = player.Osu
	// 			break
	// 		}
	// 	} else if m.Author.ID == player.Discord.ID && player.Osu.Username != "" {
	// 		user = player.Osu
	// 		break
	// 	}
	// }

	// Check if user even exists
	if user.UserID == 0 {
		if username == "" {
			ctx.Reply("No user mentioned in message/linked to your account! Please use `set` or `link` to link an osu! account to you, or name a user to obtain their recent score of!")
		}
		test, err := ctx.Osu.GetUser(osuapi.GetUserOpts{
			Username: username,
		})
		if err != nil {
			ctx.ReplyErr(err, "User %s may not exist! Are you sure you replaced spaces with `_`?", username)
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
		user = *test
	}

	// API call
	scoreOpts := osuapi.GetScoresOpts{
		BeatmapID: beatmap.BeatmapID,
		UserID:    user.UserID,
		Limit:     100,
	}
	scores, err := ctx.Osu.GetScores(scoreOpts)
	if err != nil {
		ctx.ReplyErr(err, "The osu! API just owned me. Please try again!")
		return framework.MIDDLEWARE_RESPONSE_ERR
	}
	if len(scores) == 0 {
		if username != "" {
			ctx.Reply(username + " hasn't set a score on this!")
		} else {
			ctx.Reply("You haven't set a score on this with any mod combination!")
		}
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	// Mod filter
	if mods != "NM" {
		for i := 0; i < len(scores); i++ {
			if (strict && scores[i].Mods != parsedMods) || (!strict && ((parsedMods == 0 && scores[i].Mods != 0) || scores[i].Mods&parsedMods != parsedMods)) {
				scores = append(scores[:i], scores[i+1:]...)
				i--
			}
		}
		if len(scores) == 0 {
			ctx.Reply("No scores with the mod combination **%s** exist!", mods)
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
	}
	topScore := scores[0]

	// Sort by PP
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].PP > scores[j].PP
	})

	// Get the beatmap but with mods applied if not all
	if !allRegex.MatchString(content) {
		diffMods := 338 & scores[0].Mods
		if diffMods&256 != 0 && diffMods&64 != 0 { // Remove DTHT
			diffMods -= 320
		}
		if diffMods&2 != 0 && diffMods&16 != 0 { // Remove EZHR
			diffMods -= 18
		}
		beatmap = osutools.BeatmapParse(strconv.Itoa(beatmap.BeatmapID), "map", &diffMods)
	}

	// Create embed
	// Assign timing variables for map specs
	totalMinutes := math.Floor(float64(beatmap.TotalLength / 60))
	totalSeconds := fmt.Sprint(math.Mod(float64(beatmap.TotalLength), float64(60)))
	if len(totalSeconds) == 1 {
		totalSeconds = "0" + totalSeconds
	}
	hitMinutes := math.Floor(float64(beatmap.HitLength / 60))
	hitSeconds := fmt.Sprint(math.Mod(float64(beatmap.HitLength), float64(60)))
	if len(hitSeconds) == 1 {
		hitSeconds = "0" + hitSeconds
	}
	sr := "**SR:** " + strconv.FormatFloat(beatmap.DifficultyRating, 'f', 2, 64)
	if beatmap.Mode == osuapi.ModeOsu {
		sr += " **Aim:** " + strconv.FormatFloat(beatmap.DifficultyAim, 'f', 2, 64) + " **Speed:** " + strconv.FormatFloat(beatmap.DifficultySpeed, 'f', 2, 64)
	}
	length := "**Length:** " + fmt.Sprint(totalMinutes) + ":" + fmt.Sprint(totalSeconds) + " (" + fmt.Sprint(hitMinutes) + ":" + fmt.Sprint(hitSeconds) + ") "
	bpm := "**BPM:** " + fmt.Sprint(beatmap.BPM) + " "
	mapStats := "**CS:** " + strconv.FormatFloat(beatmap.CircleSize, 'f', 1, 64) + " **AR:** " + strconv.FormatFloat(beatmap.ApproachRate, 'f', 1, 64) + " **OD:** " + strconv.FormatFloat(beatmap.OverallDifficulty, 'f', 1, 64) + " **HP:** " + strconv.FormatFloat(beatmap.HPDrain, 'f', 1, 64)
	mapObjs := "**Circles:** " + strconv.Itoa(beatmap.Circles) + " **Sliders:** " + strconv.Itoa(beatmap.Sliders) + " **Spinners:** " + strconv.Itoa(beatmap.Spinners)
	Color := osutools.ModeColour(beatmap.Mode)

	embed := &discordgo.MessageEmbed{
		Color: Color,
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://osu.ppy.sh/users/" + strconv.Itoa(user.UserID),
			Name:    user.Username,
			IconURL: "https://a.ppy.sh/" + strconv.Itoa(user.UserID) + "?" + strconv.Itoa(rand.Int()) + ".jpeg",
		},
		Description: sr + "\n" +
			length + bpm + "\n" +
			mapStats + "\n" +
			mapObjs + "\n\n",
		Title: beatmap.Artist + " - " + beatmap.Title + " [" + beatmap.DiffName + "] by " + strings.Replace(beatmap.Creator, "_", `\_`, -1),
		URL:   "https://osu.ppy.sh/beatmaps/" + strconv.Itoa(beatmap.BeatmapID),
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://b.ppy.sh/thumb/" + strconv.Itoa(beatmap.BeatmapSetID) + "l.jpg",
		},
	}
	if strings.ToLower(beatmap.Title) == "crab rave" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: "https://cdn.discordapp.com/emojis/510169818893385729.gif",
		}
	}
	for i := 0; i < len(scores); i++ {
		score := scores[i]

		// Get time since play
		timeParse, _ := time.Parse("2006-01-02 15:04:05", score.Date.String())
		time := tools.TimeSince(timeParse)

		// Assign values
		mods = score.Mods.String()
		accCalc := 100.0 * float64(score.Count50+2*score.Count100+6*score.Count300) / float64(6*(score.CountMiss+score.Count50+score.Count100+score.Count300))
		scorePrint := " **" + tools.Comma(score.Score.Score) + "** "
		acc := "** " + strconv.FormatFloat(accCalc, 'f', 2, 64) + "%** "
		hits := "**Hits:** [" + strconv.Itoa(score.Count300) + "/" + strconv.Itoa(score.Count100) + "/" + strconv.Itoa(score.Count50) + "/" + strconv.Itoa(score.CountMiss) + "]"

		replay := ""
		if score.Replay {
			replay = "| [**Replay**](https://osu.ppy.sh/scores/osu/" + strconv.FormatInt(score.ScoreID, 10) + "/download)"
			reader, _ := ctx.Osu.GetReplay(osuapi.GetReplayOpts{
				Username:  user.Username,
				Mode:      beatmap.Mode,
				BeatmapID: beatmap.BeatmapID,
				Mods:      &score.Mods,
			})
			buf := new(bytes.Buffer)
			buf.ReadFrom(reader)
			replayData := structs.ReplayData{
				Mode:    beatmap.Mode,
				Beatmap: beatmap,
				Score:   score.Score,
				Player:  user,
				Data:    buf.Bytes(),
			}
			replayData.PlayData = replayData.GetPlayData(true)
			UR := replayData.GetUnstableRate()
			replay += " | " + strconv.FormatFloat(UR, 'f', 2, 64)
			if strings.Contains(mods, "DT") || strings.Contains(mods, "NC") || strings.Contains(mods, "HT") {
				replay += " cv. UR"
			} else {
				replay += " UR"
			}
			if genOSR.MatchString(content) && ctx.MC.Author.ID == config.Conf.BotHoster.UserID {
				fileContent := replayData.CreateOSR()
				ioutil.WriteFile("./"+score.Username+strconv.Itoa(beatmap.BeatmapID)+strconv.Itoa(int(score.Mods))+".osr", fileContent, 0644)
			}
		}

		if mods == "" {
			mods = "NM"
		}

		if strings.Contains(mods, "DTNC") {
			mods = strings.Replace(mods, "DTNC", "NC", 1)
		}

		var combo string
		if score.MaxCombo == beatmap.MaxCombo {
			if accCalc == 100.0 {
				combo = " **SS** "
			} else {
				combo = " **FC** "
			}
		} else {
			combo = " **" + strconv.Itoa(score.MaxCombo) + "**/" + strconv.Itoa(beatmap.MaxCombo) + "x "
		}

		mapCompletion := ""
		if i == 0 { // Only matters for the top pp score Lol
			orderedScores, err := ctx.Osu.GetUserBest(osuapi.GetUserScoresOpts{
				Username: user.Username,
				Limit:    100,
			})
			if err != nil {
				ctx.ReplyErr(err, "The osu! API just owned me. Please try again!")
				return framework.MIDDLEWARE_RESPONSE_ERR
			}
			for i, orderedScore := range orderedScores {
				if score.Score.Score == orderedScore.Score.Score {
					mapCompletion += "**#" + strconv.Itoa(i+1) + "** in top performances! \n"
					break
				}
			}
		}
		if topScore.Score.Score == score.Score.Score { // Only matters for the top score Lol
			mapScores, err := ctx.Osu.GetScores(osuapi.GetScoresOpts{
				BeatmapID: beatmap.BeatmapID,
				Limit:     100,
			})
			if err != nil {
				ctx.ReplyErr(err, "The osu! API just owned me. Please try again!")
				return framework.MIDDLEWARE_RESPONSE_ERR
			}
			for i, mapScore := range mapScores {
				if score.UserID == mapScore.UserID && score.Score.Score == mapScore.Score.Score {
					mapCompletion += "**#" + strconv.Itoa(i+1) + "** on leaderboard! \n"
					break
				}
			}
		}

		// Get pp values
		var pp string
		totalObjs := beatmap.Circles + beatmap.Sliders + beatmap.Spinners
		if score.Score.FullCombo { // If play was a perfect combo
			pp = "**" + strconv.FormatFloat(score.PP, 'f', 2, 64) + "pp**/" + strconv.FormatFloat(score.PP, 'f', 2, 64) + "pp "
		} else { // If map was finished, but play was not a perfect combo
			ppValues := make(chan string, 1)
			go osutools.PPCalc(beatmap, osuapi.Score{
				MaxCombo: beatmap.MaxCombo,
				Count50:  score.Count50,
				Count100: score.Count100,
				Count300: totalObjs - score.Count50 - score.Count100,
				Mods:     score.Mods,
			}, ppValues)
			pp = "**" + strconv.FormatFloat(score.PP, 'f', 2, 64) + "pp**/" + <-ppValues + "pp "
		}
		mods = " **+" + mods + "** "

		score.Rank = strings.Replace(score.Rank, "X", "SS", -1)
		g, _ := ctx.S.Guild(config.Conf.Server)
		scoreRank := ""
		for _, emoji := range g.Emojis {
			if emoji.Name == score.Rank+"_" {
				scoreRank = emoji.MessageFormat()
			}
		}

		if !allRegex.MatchString(content) || len(scores) == 1 {
			embed.Description += scoreRank + scorePrint + mods + combo + acc + replay + "\n" +
				mapCompletion + "\n" +
				pp + hits + "\n\n"
			embed.Footer = &discordgo.MessageEmbedFooter{
				Text: time,
			}
			message, err := ctx.S.ChannelMessageSendEmbed(ctx.MC.ChannelID, embed)
			ctx.MC = &discordgo.MessageCreate{Message: message}
			if scorePostRegex.MatchString(content) && err == nil {
				var params []string
				if mapperRegex.MatchString(content) {
					params = append(params, "mapper")
				}
				if starRegex.MatchString(content) {
					params = append(params, "sr")
				}
				ScorePost(ctx, "", params...)
			}
			return framework.MIDDLEWARE_RESPONSE_OK
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "#" + strconv.Itoa(i+1) + " | " + time,
			Value: scoreRank + scorePrint + mods + combo + acc + replay + "\n" +
				mapCompletion +
				pp + hits + "\n\n",
		})
		if (i+1)%25 == 0 {
			ctx.S.ChannelMessageSendEmbed(ctx.MC.ChannelID, embed)
			embed.Fields = []*discordgo.MessageEmbedField{}
		}
	}
	if len(scores)%25 != 0 {
		ctx.S.ChannelMessageSendEmbed(ctx.MC.ChannelID, embed)
	}
	return framework.MIDDLEWARE_RESPONSE_OK
}

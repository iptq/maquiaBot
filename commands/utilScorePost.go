package commands

import (
	"bytes"
	"image/png"
	"io/ioutil"
	"maquiaBot/framework"
	osuapi "maquiaBot/osu-api"
	osutools "maquiaBot/osu-tools"
	"maquiaBot/structs"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	mapRegex         = regexp.MustCompile(`(?i)(https:\/\/)?(osu|old)\.ppy\.sh\/(s|b|beatmaps|beatmapsets)\/(\d+)(#(osu|taiko|fruits|mania)\/(\d+))?`)
	scorePostRegex   = regexp.MustCompile(`(?i)sc?(orepost)?\s+(\S+)`)
	spModRegex       = regexp.MustCompile(`(?i)-m\s+(\S+)`)
	mod2Regex        = regexp.MustCompile(`(?i)\+(\S+)`)
	scoreRegex       = regexp.MustCompile(`(?i)\*\*(([0-9]|,)+)\*\* `)
	leaderboardRegex = regexp.MustCompile(`(?i)\*\*(#\d+)\*\* on leaderboard!`)
	spMapperRegex    = regexp.MustCompile(`(?i)-mapper`)
	spStarRegex      = regexp.MustCompile(`(?i)-sr`)
)

// ScorePost posts your score in a single line format
func ScorePost(ctx *framework.CommandContext, postType string, params ...string) {

	var beatmap osuapi.Beatmap
	var username string
	var user osuapi.User
	mods := "NM"
	parsedMods := osuapi.Mods(0)
	leaderboard := ""
	scoreVal := int64(0)
	content := ctx.MC.Content

	if postType == "scorePost" {
		if scorePostRegex.MatchString(content) {
			username = scorePostRegex.FindStringSubmatch(content)[2]
		}
		if spModRegex.MatchString(username) {
			mods = strings.ToUpper(spModRegex.FindStringSubmatch(username)[1])
			if strings.Contains(mods, "NC") && !strings.Contains(mods, "DT") {
				mods += "DT"
			}
			parsedMods = osuapi.ParseMods(mods)

			username = strings.TrimSpace(strings.Replace(username, spModRegex.FindStringSubmatch(username)[0], "", 1))
		}
		if spMapperRegex.MatchString(content) {
			username = strings.TrimSpace(strings.Replace(username, spMapperRegex.FindStringSubmatch(username)[0], "", 1))
		}
		if spStarRegex.MatchString(content) {
			username = strings.TrimSpace(strings.Replace(username, spStarRegex.FindStringSubmatch(username)[0], "", 1))
		}

		// Get the map
		var submatches []string
		parsed := false
		if mapRegex.MatchString(content) {
			submatches = mapRegex.FindStringSubmatch(content)
		} else {
			// Get prev messages
			messages, err := ctx.S.ChannelMessages(ctx.MC.ChannelID, -1, "", "", "")
			if err != nil {
				ctx.ReplyErr(err, "No map to compare to!")
				return
			}

			// Look for a valid beatmap ID
			for _, msg := range messages {
				if len(msg.Embeds) > 0 && msg.Embeds[0].Author != nil {
					if mapRegex.MatchString(msg.Embeds[0].URL) {
						submatches = mapRegex.FindStringSubmatch(msg.Embeds[0].URL)
						if username == "" && mods == "" && mapRegex.FindStringSubmatch(ctx.MC.Embeds[0].URL)[3] == "beatmaps" && ctx.MC.Author.ID == ctx.S.State.User.ID {
							nomod := osuapi.Mods(0)
							beatmap = osutools.BeatmapParse(mapRegex.FindStringSubmatch(ctx.MC.Embeds[0].URL)[4], "map", &nomod)

							username = ctx.MC.Embeds[0].Author.Name
							test, err := ctx.Osu.GetUser(osuapi.GetUserOpts{
								Username: username,
							})
							if err != nil {
								ctx.ReplyErr(err, "User %s may not exist! Are you sure you replaced spaces with `_`?", username)
								return
							}
							user = *test

							mods = mod2Regex.FindStringSubmatch(ctx.MC.Embeds[0].Description)[1]
							if strings.Contains(mods, "NC") && !strings.Contains(mods, "DT") {
								mods += "DT"
							}
							parsedMods = osuapi.ParseMods(mods)

							scoreText := strings.Replace(scoreRegex.FindStringSubmatch(ctx.MC.Embeds[0].Description)[1], ",", "", -1)
							scoreVal, _ = strconv.ParseInt(scoreText, 10, 64)

							if leaderboardRegex.MatchString(ctx.MC.Embeds[0].Description) {
								leaderboard = leaderboardRegex.FindStringSubmatch(ctx.MC.Embeds[0].Description)[1] + " "
							} else {
								leaderboard = "N/A"
							}
							parsed = true
						}
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

		if !parsed {
			// Check if found
			if len(submatches) == 0 {
				ctx.Reply("No map to create a score post for!")
				return
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
				ctx.Reply("No map to create a score post for!")
				return
			} else if beatmap.Approved < 1 {
				ctx.Reply("The map `%s - %s` does not have a leaderboard!", beatmap.Artist, beatmap.Title)
				return
			}
			username = strings.TrimSpace(strings.Replace(username, submatches[0], "", -1))

			// Get user
			player, err := ctx.GetOsuProfile()
			if err != nil {
				ctx.ReplyErr(err, "couldn't get osu user")
				return
			} else {
				user = player.Osu
			}

			// for _, player := range cache {
			// 	if username != "" {
			// 		if username == player.Osu.Username {
			// 			user = player.Osu
			// 			break
			// 		}
			// 	} else if ctx.MC.Author.ID == player.Discord.ID && player.Osu.Username != "" {
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
					return
				}
				user = *test
			}
		}
	} else {
		nomod := osuapi.Mods(0)
		if mapRegex.MatchString(ctx.MC.Embeds[0].URL) {
			beatmap = osutools.BeatmapParse(mapRegex.FindStringSubmatch(ctx.MC.Embeds[0].URL)[4], "map", &nomod)
		}

		username = ctx.MC.Embeds[0].Author.Name
		test, err := ctx.Osu.GetUser(osuapi.GetUserOpts{
			Username: username,
		})
		if err != nil {
			ctx.ReplyErr(err, "User %s may not exist! Are you sure you replaced spaces with `_`?", username)
			return
		}
		user = *test

		mods = mod2Regex.FindStringSubmatch(ctx.MC.Embeds[0].Description)[1]
		if strings.Contains(mods, "NC") && !strings.Contains(mods, "DT") {
			mods += "DT"
		}
		parsedMods = osuapi.ParseMods(mods)

		scoreText := strings.Replace(scoreRegex.FindStringSubmatch(ctx.MC.Embeds[0].Description)[1], ",", "", -1)
		scoreVal, _ = strconv.ParseInt(scoreText, 10, 64)

		if leaderboardRegex.MatchString(ctx.MC.Embeds[0].Description) {
			leaderboard = leaderboardRegex.FindStringSubmatch(ctx.MC.Embeds[0].Description)[1] + " "
		} else {
			leaderboard = "N/A"
		}
	}

	// API call
	var score osuapi.Score
	var replayData structs.ReplayData
	if postType == "recent" || postType == "recentBest" {
		replayScore, err := ctx.Osu.GetScores(osuapi.GetScoresOpts{
			BeatmapID: beatmap.BeatmapID,
			UserID:    user.UserID,
			Mods:      &parsedMods,
		})
		if err == nil && len(replayScore) > 0 {
			score = replayScore[0].Score

			if score.Score != scoreVal {
				scoreOpts := osuapi.GetUserScoresOpts{
					UserID: user.UserID,
					Limit:  50,
				}
				scores, err := ctx.Osu.GetUserRecent(scoreOpts)
				if err != nil {
					ctx.ReplyErr(err, "The osu! API just owned me. Please try again!")
					return
				}
				if len(scores) == 0 {
					ctx.Reply("Could not create a scorepost for the score above!")
					return
				}
				for _, recentScore := range scores {
					if recentScore.Score.Score == scoreVal {
						score = recentScore.Score
						break
					}
				}

				if score.Score != scoreVal {
					ctx.Reply("Could not create a scorepost for the score above!")
					return
				}
			}
		}

	} else if postType != "" {
		res, err := http.Get(postType)
		if err != nil {
			ctx.Reply("Could not create a scorepost for the score above!")
			return
		}
		defer res.Body.Close()

		replayInfo, err := ioutil.ReadAll(res.Body)
		if err != nil || len(replayInfo) <= 81 {
			ctx.ReplyErr(err, "Could not create a scorepost for the score above! Replay does not have enough information.")
			return
		}

		// Parse replay data
		replayData = structs.ReplayData{
			Data: replayInfo,
		}
		replayData.ParseReplay(ctx.Osu)
		if replayData.Beatmap.BeatmapID != 0 {
			diffMods := osuapi.Mods(338) & replayData.Score.Mods
			replayData.Beatmap = osutools.BeatmapParse(strconv.Itoa(replayData.Beatmap.BeatmapID), "map", &diffMods)
		}
		replayData.UnstableRate = replayData.GetUnstableRate()
		score = replayData.Score
		beatmap = replayData.Beatmap
	} else {
		scoreOpts := osuapi.GetScoresOpts{
			BeatmapID: beatmap.BeatmapID,
			UserID:    user.UserID,
		}
		if parsedMods != 0 {
			scoreOpts.Mods = &parsedMods
		}
		scores, err := ctx.Osu.GetScores(scoreOpts)
		if err != nil {
			ctx.ReplyErr(err, "The osu! API just owned me. Please try again!")
			return
		}
		if len(scores) == 0 {
			ctx.Reply("Could not create a scorepost for the score above!")
			return
		}
		score = scores[0].Score
	}

	if replayData.UnstableRate == 0 {
		diffMods := 338 & score.Mods
		if diffMods&256 != 0 && diffMods&64 != 0 { // Remove DTHT
			diffMods -= 320
		}
		if diffMods&2 != 0 && diffMods&16 != 0 { // Remove EZHR
			diffMods -= 18
		}
		beatmap = osutools.BeatmapParse(strconv.Itoa(beatmap.BeatmapID), "map", &diffMods)
	}

	accCalc := 100.0 * float64(score.Count50+2*score.Count100+6*score.Count300) / float64(6*(score.CountMiss+score.Count50+score.Count100+score.Count300))
	acc := strconv.FormatFloat(accCalc, 'f', 2, 64) + "%"

	text := user.Username + " | " +
		beatmap.Artist + " - " + beatmap.Title +
		" [" + beatmap.DiffName + "] +"

	if beatmap.Artist == "" {
		text = user.Username + " | " +
			beatmap.Title +
			" [" + beatmap.DiffName + "] +"
	}

	modText := strings.Replace(score.Mods.String(), "DTNC", "NC", -1)
	newModText := ""
	for i := range modText {
		newModText += string(modText[i])
		if i > 0 && (i-1)%2 == 0 && i != len(modText)-1 {
			newModText += ","
		}
	}
	text += newModText
	if accCalc != 100.0 {
		text += " (" + acc + ")"
	}

	mapper := true
	sr := true
	for _, param := range params {
		if param == "mapper" {
			mapper = false
		} else if param == "sr" {
			sr = false
		}
	}

	mapperSR := " (" + strings.Replace(beatmap.Creator, "_", `\_`, -1) + " | " + strconv.FormatFloat(beatmap.DifficultyRating, 'f', 2, 64) + "★)"
	if spStarRegex.MatchString(content) || !sr {
		mapperSR = " (mapset by " + strings.Replace(beatmap.Creator, "_", `\_`, -1) + ")"
	}
	if spMapperRegex.MatchString(content) || !mapper {
		mapperSR = " " + strconv.FormatFloat(beatmap.DifficultyRating, 'f', 2, 64) + "★"
	}
	if (spStarRegex.MatchString(content) && spMapperRegex.MatchString(content)) || (!mapper && !sr) {
		mapperSR = ""
	}
	text += mapperSR

	text = strings.Replace(text, " +NM", "", -1)

	if score.MaxCombo == beatmap.MaxCombo {
		if accCalc == 100.0 {
			text += " SS "
		} else {
			text += " FC "
		}
	} else {
		if score.CountMiss == 0 {
			text += " " + strconv.Itoa(score.MaxCombo) + "/" + strconv.Itoa(beatmap.MaxCombo) + "x "
		} else {
			text += " " + strconv.Itoa(score.CountMiss) + "m " + strconv.Itoa(score.MaxCombo) + "/" + strconv.Itoa(beatmap.MaxCombo) + "x "
		}
	}

	if leaderboard == "" {
		leaderboardScores, _ := ctx.Osu.GetScores(osuapi.GetScoresOpts{
			BeatmapID: beatmap.BeatmapID,
			Limit:     50,
		})
		for i, mapScore := range leaderboardScores {
			if score.UserID == mapScore.UserID && score.Score == mapScore.Score.Score {
				text += "#" + strconv.Itoa(i+1) + " "
				break
			}
		}
	} else if leaderboard != "N/A" {
		text += leaderboard
	}

	ppValues := make(chan string, 1)
	go osutools.PPCalc(beatmap, score, ppValues)
	ppVal, _ := strconv.ParseFloat(<-ppValues, 64)
	if beatmap.Approved == osuapi.StatusLoved {
		text += "LOVED | "
	} else if beatmap.Approved == osuapi.StatusQualified {
		text += "QUALIFIED | "
	} else {
		text += "| "
	}

	text += strconv.FormatFloat(ppVal, 'f', 0, 64)

	if beatmap.Approved != osuapi.StatusRanked && beatmap.Approved != osuapi.StatusApproved {
		text += "pp if ranked | "
	} else {
		text += "pp | "
	}

	if replayData.UnstableRate != 0 {
		text += strconv.FormatFloat(replayData.UnstableRate, 'f', 2, 64)
		score.Replay = true
		if score.Mods&256 != 0 || score.Mods&64 != 0 {
			text += " cv. UR"
		} else {
			text += " UR"
		}
	} else if score.Replay {
		reader, _ := ctx.Osu.GetReplay(osuapi.GetReplayOpts{
			Username:  user.Username,
			Mode:      beatmap.Mode,
			BeatmapID: beatmap.BeatmapID,
			Mods:      &score.Mods,
		})
		buf := new(bytes.Buffer)
		buf.ReadFrom(reader)
		replayData = structs.ReplayData{
			Mode:    beatmap.Mode,
			Beatmap: beatmap,
			Score:   score,
			Data:    buf.Bytes(),
		}
		replayData.PlayData = replayData.GetPlayData(true)
		UR := replayData.GetUnstableRate()
		text += strconv.FormatFloat(UR, 'f', 2, 64)
		score.Replay = true
		if score.Mods&256 != 0 || score.Mods&64 != 0 {
			text += " cv. UR"
		} else {
			text += " UR"
		}
		replayData.UnstableRate = UR
	}

	ctx.S.ChannelMessageSend(ctx.MC.ChannelID, text)

	img, err := osutools.ResultImage(score, beatmap, user, replayData)
	if err != nil {
		ctx.ReplyErr(err, "Failed to create scorepost image")
		return
	} else {
		imgBytes := new(bytes.Buffer)
		_ = png.Encode(imgBytes, img)
		ctx.S.ChannelMessageSendComplex(ctx.MC.ChannelID, &discordgo.MessageSend{
			Files: []*discordgo.File{
				&discordgo.File{
					Name:   "image.png",
					Reader: imgBytes,
				},
			},
		})
	}
}

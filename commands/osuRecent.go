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

var (
	recentRegex          = regexp.MustCompile(`(?i)(r|recent|rs|rb|recentb|recentbest)\s+(.+)`)
	modRegex             = regexp.MustCompile(`(?i)-m\s+(\S+)`)
	strictRegex          = regexp.MustCompile(`(?i)-nostrict`)
	recentScorePostRegex = regexp.MustCompile(`(?i)-sp`)
	mapperRegex          = regexp.MustCompile(`(?i)-mapper`)
	starRegex            = regexp.MustCompile(`(?i)-sr`)
	genOSR               = regexp.MustCompile(`(?i)-osr`)
)

type _Recent struct {
}

func Recent() _Recent {
	return _Recent{}
}

func (m _Recent) Help(embed *discordgo.MessageEmbed) {
	embed.Author.Name = "Command: r / rs / recent"
	embed.Description = "`(r|rs|recent) [osu! username] [num] [-m mod] [-sp [-mapper] [-sr]]` shows the player's recent score."
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "[osu! username]",
			Value:  "The osu! user to check. No user given will use the account linked to your discord account.",
			Inline: true,
		},
		{
			Name:   "[num]",
			Value:  "The nth recent score to find (Default: Latest).",
			Inline: true,
		},
		{
			Name:   "[-m mod]",
			Value:  "The mods to check for.",
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
		{
			Name:  "Related Commands:",
			Value: "`recentbest`",
		},
	}
}

func (m _Recent) Handle(ctx *framework.CommandContext) int {
	return handleRecent(ctx, "recent")
}

type _RecentBest struct {
}

func RecentBest() _RecentBest {
	return _RecentBest{}
}

func (m _RecentBest) Help(embed *discordgo.MessageEmbed) {
	embed.Author.Name = "Command: rb / recentb / recentbest"
	embed.Description = "`(rb|recentb|recentbest) [osu! username] [num] [-m mod] [-sp [-mapper] [-sr]]` shows the player's recent top 100 pp score."
	embed.Fields = []*discordgo.MessageEmbedField{
		{
			Name:   "[osu! username]",
			Value:  "The osu! user to check. No user given will use the account linked to your discord account.",
			Inline: true,
		},
		{
			Name:   "[num]",
			Value:  "The nth recent top score to find (Default: Latest).",
			Inline: true,
		},
		{
			Name:   "[-m mod]",
			Value:  "The mods to check for.",
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
		{
			Name:  "Related Commands:",
			Value: "`recentbest`",
		},
	}
}

func (m _RecentBest) Handle(ctx *framework.CommandContext) int {
	return handleRecent(ctx, "best")
}

func handleRecent(ctx *framework.CommandContext, option string) int {
	username := ""
	mods := ""
	index := 1
	strict := true
	content := ctx.MC.Content

	// Obtain index, mods, strict, and username
	if recentRegex.MatchString(content) {
		username = recentRegex.FindStringSubmatch(content)[2]
		if modRegex.MatchString(username) {
			mods = strings.ToUpper(modRegex.FindStringSubmatch(username)[1])
			if strings.Contains(mods, "NC") && !strings.Contains(mods, "DT") {
				mods += "DT"
			}
			username = strings.TrimSpace(strings.Replace(username, modRegex.FindStringSubmatch(username)[0], "", 1))
		}

		if strictRegex.MatchString(content) {
			strict = false
			username = strings.TrimSpace(strings.Replace(username, strictRegex.FindStringSubmatch(content)[0], "", 1))
		}
		if recentScorePostRegex.MatchString(content) {
			username = strings.TrimSpace(strings.Replace(username, recentScorePostRegex.FindStringSubmatch(content)[0], "", 1))
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

		usernameSplit := strings.Split(username, " ")
		for _, txt := range usernameSplit {
			if i, err := strconv.Atoi(txt); err == nil && i > 0 && i <= 100 {
				username = strings.TrimSpace(strings.Replace(username, txt, "", 1))
				index = i
				break
			}
		}
	}

	// If the user specified a name, use it.
	// If not, find the user's name and return it
	var userP *osuapi.User
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
			userP = &player.Osu
			username = player.Osu.Username
		}
	} else {
		userP, err = ctx.Osu.GetUser(osuapi.GetUserOpts{
			Username: username,
		})
		if err != nil {
			ctx.ReplyErr(err, "User **%s** may not exist!")
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
	}

	// Run api call for user best/recent
	var scoreList []osuapi.GUSScore
	switch option {
	case "recent":
		scoreList, err = ctx.Osu.GetUserRecent(osuapi.GetUserScoresOpts{
			Username: username,
			Limit:    50,
		})
		if err != nil {
			ctx.ReplyErr(err, "Couldn't retrieve user recent")
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
		if len(scoreList) == 0 {
			ctx.Reply("%s has not played recently!", username)
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
	case "best":
		scoreList, err = ctx.Osu.GetUserBest(osuapi.GetUserScoresOpts{
			Username: username,
			Limit:    100,
		})
		if err != nil {
			ctx.ReplyErr(err, "Couldn't retrieve user best")
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
		if len(scoreList) == 0 {
			ctx.Reply("%s has no top scores!", username)
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
	}

	// Sort scores by date and get score
	err = osuapi.DateSortGUS(scoreList)
	if err != nil {
		ctx.ReplyErr(err, "can't sort invalid date format")
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	// Mod filter
	if mods != "" {
		parsedMods := osuapi.ParseMods(mods)
		for i := 0; i < len(scoreList); i++ {
			if (strict && scoreList[i].Mods != parsedMods) || (!strict && ((parsedMods == 0 && scoreList[i].Mods != 0) || scoreList[i].Mods&parsedMods != parsedMods)) {
				scoreList = append(scoreList[:i], scoreList[i+1:]...)
				i--
			}
		}
		if len(scoreList) == 0 {
			ctx.Reply("No scores with the mod combination **%s** exist!", mods)
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
	}

	warning := ""
	if index > len(scoreList) {
		index = len(scoreList)
		warning = "Defaulted to max: " + strconv.Itoa(len(scoreList))
	}
	score := scoreList[index-1]

	// Get beatmap, acc, and mods
	diffMods := osuapi.Mods(338) & score.Mods
	beatmap := osutools.BeatmapParse(strconv.Itoa(score.BeatmapID), "map", &diffMods)
	accCalc := 100.0 * float64(score.Count50+2*score.Count100+6*score.Count300) / float64(6*(score.CountMiss+score.Count50+score.Count100+score.Count300))
	mods = score.Mods.String()

	// Count number of tries
	try := 0
	for i := index - 1; i < len(scoreList); i++ {
		if scoreList[i].BeatmapID == score.BeatmapID {
			try++
		} else {
			break
		}
	}

	// Count number of objs
	objCount := beatmap.Circles + beatmap.Sliders + beatmap.Spinners
	playObjCount := score.CountMiss + score.Count100 + score.Count300 + score.Count50

	// Get time since play
	timeParse, _ := time.Parse("2006-01-02 15:04:05", score.Date.String())

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

	// Assign misc variables
	Color := osutools.ModeColour(beatmap.Mode)
	sr := "**SR:** " + strconv.FormatFloat(beatmap.DifficultyRating, 'f', 2, 64)
	if beatmap.Mode == osuapi.ModeOsu {
		sr += " **Aim:** " + strconv.FormatFloat(beatmap.DifficultyAim, 'f', 2, 64) + " **Speed:** " + strconv.FormatFloat(beatmap.DifficultySpeed, 'f', 2, 64)
	}
	length := "**Length:** " + fmt.Sprint(totalMinutes) + ":" + fmt.Sprint(totalSeconds) + " (" + fmt.Sprint(hitMinutes) + ":" + fmt.Sprint(hitSeconds) + ") "
	bpm := "**BPM:** " + fmt.Sprint(beatmap.BPM) + " "
	mapStats := "**CS:** " + strconv.FormatFloat(beatmap.CircleSize, 'f', 1, 64) + " **AR:** " + strconv.FormatFloat(beatmap.ApproachRate, 'f', 1, 64) + " **OD:** " + strconv.FormatFloat(beatmap.OverallDifficulty, 'f', 1, 64) + " **HP:** " + strconv.FormatFloat(beatmap.HPDrain, 'f', 1, 64)
	mapObjs := "**Circles:** " + strconv.Itoa(beatmap.Circles) + " **Sliders:** " + strconv.Itoa(beatmap.Sliders) + " **Spinners:** " + strconv.Itoa(beatmap.Spinners)
	scorePrint := " **" + tools.Comma(score.Score.Score) + "** "
	var combo string
	var mapCompletion string

	if strings.Contains(mods, "DTNC") {
		mods = strings.Replace(mods, "DTNC", "NC", 1)
	}

	if score.MaxCombo == beatmap.MaxCombo {
		if accCalc == 100.0 {
			combo = " **SS** "
		} else {
			combo = " **FC** "
		}
	} else {
		combo = " **" + strconv.Itoa(score.MaxCombo) + "**/" + strconv.Itoa(beatmap.MaxCombo) + "x "
	}

	if objCount != playObjCount {
		completed := float64(playObjCount) / float64(objCount) * 100.0
		mapCompletion = "**" + strconv.FormatFloat(completed, 'f', 2, 64) + "%** completed \n"
	} else {
		orderedScores, err := ctx.Osu.GetUserBest(osuapi.GetUserScoresOpts{
			Username: userP.Username,
			Limit:    100,
		})
		if err != nil {
			ctx.ReplyErr(err, "Couldn't retrieve user best")
			return framework.MIDDLEWARE_RESPONSE_ERR
		}
		for i, orderedScore := range orderedScores {
			if score.Score.Score == orderedScore.Score.Score {
				mapCompletion += "**#" + strconv.Itoa(i+1) + "** in top performances! \n"
				break
			}
		}
		mapScores, err := ctx.Osu.GetScores(osuapi.GetScoresOpts{
			BeatmapID: beatmap.BeatmapID,
			Limit:     100,
		})
		if err != nil {
			ctx.ReplyErr(err, "Couldn't retrieve scores")
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
	if score.PP == 0 { // If map was not finished
		ppValues := make(chan string, 2)
		var ppValueArray [2]string
		go osutools.PPCalc(beatmap, osuapi.Score{
			MaxCombo: beatmap.MaxCombo,
			Count50:  score.Count50,
			Count100: score.Count100,
			Count300: totalObjs - score.Count50 - score.Count100,
			Mods:     score.Mods,
		}, ppValues)
		go osutools.PPCalc(beatmap, score.Score, ppValues)
		for v := 0; v < 2; v++ {
			ppValueArray[v] = <-ppValues
		}
		sort.Slice(ppValueArray[:], func(i, j int) bool {
			pp1, _ := strconv.ParseFloat(ppValueArray[i], 64)
			pp2, _ := strconv.ParseFloat(ppValueArray[j], 64)
			return pp1 > pp2
		})
		if objCount != playObjCount {
			pp = "~~**" + ppValueArray[1] + "pp**~~/" + ppValueArray[0] + "pp "
		} else {
			pp = "**" + ppValueArray[1] + "pp**/" + ppValueArray[0] + "pp "
		}
	} else if score.Score.FullCombo { // If play was a perfect combo
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
	acc := "** " + strconv.FormatFloat(accCalc, 'f', 2, 64) + "%** "
	hits := "**Hits:** [" + strconv.Itoa(score.Count300) + "/" + strconv.Itoa(score.Count100) + "/" + strconv.Itoa(score.Count50) + "/" + strconv.Itoa(score.CountMiss) + "]"
	mods = " **+" + mods + "** "

	replay := ""
	if score.Replay {
		replay = "| [**Replay**](https://osu.ppy.sh/scores/osu/" + strconv.FormatInt(score.ScoreID, 10) + "/download)"
		reader, _ := ctx.Osu.GetReplay(osuapi.GetReplayOpts{
			Username:  userP.Username,
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
			Player:  *userP,
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
			ioutil.WriteFile("./"+userP.Username+strconv.Itoa(beatmap.BeatmapID)+strconv.Itoa(int(score.Mods))+".osr", fileContent, 0644)
		}
	} else if option == "recent" && objCount == playObjCount {
		replayScore, err := ctx.Osu.GetScores(osuapi.GetScoresOpts{
			BeatmapID: beatmap.BeatmapID,
			UserID:    userP.UserID,
			Mods:      &score.Mods,
		})
		if err == nil && len(replayScore) > 0 && replayScore[0].Replay && replayScore[0].Score.Score == score.Score.Score {
			replay = "| [**Replay**](https://osu.ppy.sh/scores/osu/" + strconv.FormatInt(replayScore[0].ScoreID, 10) + "/download)"
			reader, _ := ctx.Osu.GetReplay(osuapi.GetReplayOpts{
				Username:  userP.Username,
				Mode:      beatmap.Mode,
				BeatmapID: beatmap.BeatmapID,
				Mods:      &score.Mods,
			})
			buf := new(bytes.Buffer)
			buf.ReadFrom(reader)
			replayData := structs.ReplayData{
				Mode:    beatmap.Mode,
				Beatmap: beatmap,
				Score:   replayScore[0].Score,
				Player:  *userP,
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
		}
	}

	score.Rank = strings.Replace(score.Rank, "X", "SS", -1)
	g, err := ctx.S.Guild(config.Conf.Server)
	if err != nil {
		ctx.ReplyErr(err, "could not retrieve guild info")
		return framework.MIDDLEWARE_RESPONSE_ERR
	}

	scoreRank := ""
	for _, emoji := range g.Emojis {
		if emoji.Name == score.Rank+"_" {
			scoreRank = emoji.MessageFormat()
		}
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Color: Color,
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://osu.ppy.sh/users/" + strconv.Itoa(userP.UserID),
			Name:    userP.Username,
			IconURL: "https://a.ppy.sh/" + strconv.Itoa(userP.UserID) + "?" + strconv.Itoa(rand.Int()) + ".jpeg",
		},
		Title: beatmap.Artist + " - " + beatmap.Title + " [" + beatmap.DiffName + "] by " + strings.Replace(beatmap.Creator, "_", `\_`, -1),
		URL:   "https://osu.ppy.sh/beatmaps/" + strconv.Itoa(beatmap.BeatmapID),
		Description: sr + "\n" +
			length + bpm + "\n" +
			mapStats + "\n" +
			mapObjs + "\n\n" +
			scoreRank + scorePrint + mods + combo + acc + replay + "\n" +
			mapCompletion + "\n" +
			pp + hits + "\n\n",
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://b.ppy.sh/thumb/" + strconv.Itoa(beatmap.BeatmapSetID) + "l.jpg",
		},
	}
	if strings.ToLower(beatmap.Title) == "crab rave" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: "https://cdn.discordapp.com/emojis/510169818893385729.gif",
		}
	}
	if option == "best" {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: tools.TimeSince(timeParse),
		}
	} else if option == "recent" {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: "Try #" + strconv.Itoa(try) + " | " + tools.TimeSince(timeParse),
		}
	}
	m, err := ctx.S.ChannelMessageSendComplex(ctx.MC.ChannelID, &discordgo.MessageSend{
		Content: warning,
		Embed:   embed,
	})
	ctx.MC = &discordgo.MessageCreate{Message: m}

	if recentScorePostRegex.MatchString(content) && err == nil {
		var params []string
		if mapperRegex.MatchString(content) {
			params = append(params, "mapper")
		}
		if starRegex.MatchString(content) {
			params = append(params, "sr")
		}
		if option == "best" {
			ScorePost(ctx, "recentBest", params...)
		} else if option == "recent" {
			ScorePost(ctx, "recent", params...)
		}
	}
	return framework.MIDDLEWARE_RESPONSE_ERR
}

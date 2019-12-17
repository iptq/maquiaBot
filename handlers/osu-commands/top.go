package osucommands

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	config "../../config"
	osuapi "../../osu-api"
	osutools "../../osu-functions"
	structs "../../structs"
	tools "../../tools"
	"github.com/bwmarrin/discordgo"
)

// Top gets the nth top pp score
func Top(s *discordgo.Session, m *discordgo.MessageCreate, cache []structs.PlayerData, mapCache []structs.MapData) {
	topRegex, _ := regexp.Compile(`t(op)?\s+(.+)`)
	modRegex, _ := regexp.Compile(`-m\s*(\S{2,})`)
	strictRegex, _ := regexp.Compile(`-nostrict`)

	username := ""
	mods := ""
	index := 1
	strict := true

	// Obtain index, mods, strict, and username
	if topRegex.MatchString(m.Content) {
		username = topRegex.FindStringSubmatch(m.Content)[2]
		if modRegex.MatchString(username) {
			mods = strings.ToUpper(modRegex.FindStringSubmatch(username)[1])
			if strings.Contains(mods, "NC") && !strings.Contains(mods, "DT") {
				mods += "DT"
			}
			username = strings.TrimSpace(strings.Replace(username, modRegex.FindStringSubmatch(username)[0], "", 1))
		}
		usernameSplit := strings.Split(username, " ")
		for _, txt := range usernameSplit {
			if i, err := strconv.Atoi(txt); err == nil && i > 0 && i <= 100 {
				username = strings.TrimSpace(strings.Replace(username, txt, "", 1))
				index = i
				break
			}
		}
		if strictRegex.MatchString(m.Content) {
			strict = false
			username = strings.TrimSpace(strings.Replace(username, strictRegex.FindStringSubmatch(m.Content)[0], "", 1))
		}
	}

	// Get message author's osu! user if no user was specified
	if username == "" {
		for _, player := range cache {
			if m.Author.ID == player.Discord.ID && player.Osu.Username != "" {
				username = player.Osu.Username
				break
			}
		}
		if username == "" {
			s.ChannelMessageSend(m.ChannelID, "Could not find any osu! account linked for "+m.Author.Mention()+" ! Please use `set` or `link` to link an osu! account to you!")
			return
		}
	}
	user, err := OsuAPI.GetUser(osuapi.GetUserOpts{
		Username: username,
	})
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "User **"+username+"** may not exist!")
		return
	}
	score := osuapi.GUSScore{}

	// Get best scores
	scoreList, err := OsuAPI.GetUserBest(osuapi.GetUserScoresOpts{
		Username: username,
		Limit:    100,
	})
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "The osu! API just owned me. Please try again!")
		return
	}
	if len(scoreList) == 0 {
		s.ChannelMessageSend(m.ChannelID, username+" has no top scores!")
		return
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
			s.ChannelMessageSend(m.ChannelID, "No scores with the mod combination **"+mods+"** exist in your top plays!")
			return
		}
	}

	warning := ""
	if index > len(scoreList) {
		index = len(scoreList)
		warning = "Defaulted to max: " + strconv.Itoa(len(scoreList))
	}
	score = scoreList[index-1]

	// Get beatmap, acc, and mods
	beatmap := osutools.BeatmapParse(strconv.Itoa(score.BeatmapID), "map", score.Mods)
	accCalc := (50.0*float64(score.Count50) + 100.0*float64(score.Count100) + 300.0*float64(score.Count300)) / (300.0 * float64(score.CountMiss+score.Count50+score.Count100+score.Count300)) * 100.0
	mods = score.Mods.String()
	if mods == "" {
		mods = "NM"
	}

	// Get time since play
	timeParse, err := time.Parse("2006-01-02 15:04:05", score.Date.String())
	tools.ErrRead(err)
	time := tools.TimeSince(timeParse)

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
	sr, _, _, _, _, _ := osutools.BeatmapCache(mods, beatmap, mapCache)
	length := "**Length:** " + fmt.Sprint(totalMinutes) + ":" + fmt.Sprint(totalSeconds) + " (" + fmt.Sprint(hitMinutes) + ":" + fmt.Sprint(hitSeconds) + ") "
	bpm := "**BPM:** " + fmt.Sprint(beatmap.BPM) + " "
	mapStats := "**CS:** " + strconv.FormatFloat(beatmap.CircleSize, 'f', 1, 64) + " **AR:** " + strconv.FormatFloat(beatmap.ApproachRate, 'f', 1, 64) + " **OD:** " + strconv.FormatFloat(beatmap.OverallDifficulty, 'f', 1, 64) + " **HP:** " + strconv.FormatFloat(beatmap.HPDrain, 'f', 1, 64)
	mapObjs := "**Circles:** " + strconv.Itoa(beatmap.Circles) + " **Sliders:** " + strconv.Itoa(beatmap.Sliders) + " **Spinners:** " + strconv.Itoa(beatmap.Spinners)
	scorePrint := " **" + tools.Comma(score.Score.Score) + "** "

	if strings.Contains(mods, "DTNC") {
		mods = strings.Replace(mods, "DTNC", "NC", 1)
	}
	scoreMods := mods
	mods = " **+" + mods + "** "

	var combo string
	if score.MaxCombo == beatmap.MaxCombo {
		if accCalc == 100.0 {
			combo = " **SS** "
		} else {
			combo = " **FC** "
		}
	} else {
		combo = " **x" + strconv.Itoa(score.MaxCombo) + "**/" + strconv.Itoa(beatmap.MaxCombo) + " "
	}

	mapCompletion := ""
	orderedScores, err := OsuAPI.GetUserBest(osuapi.GetUserScoresOpts{
		Username: user.Username,
		Limit:    100,
	})
	tools.ErrRead(err)
	for i, orderedScore := range orderedScores {
		if score.Score.Score == orderedScore.Score.Score {
			mapCompletion += "**#" + strconv.Itoa(i+1) + "** in top performances! \n"
			break
		}
	}
	mapScores, err := OsuAPI.GetScores(osuapi.GetScoresOpts{
		BeatmapID: beatmap.BeatmapID,
		Limit:     100,
	})
	tools.ErrRead(err)
	for i, mapScore := range mapScores {
		if score.UserID == mapScore.UserID && score.Score.Score == mapScore.Score.Score {
			mapCompletion += "**#" + strconv.Itoa(i+1) + "** on leaderboard! \n"
			break
		}
	}

	// Get pp values
	var pp string
	totalObjs := beatmap.Circles + beatmap.Sliders + beatmap.Spinners
	if score.Score.FullCombo { // If play was a perfect combo
		pp = "**" + strconv.FormatFloat(score.PP, 'f', 0, 64) + "pp**/" + strconv.FormatFloat(score.PP, 'f', 0, 64) + "pp "
	} else { // If play wasn't a perfect combo
		ppValues := make(chan string, 1)
		accCalcNoMiss := (50.0*float64(score.Count50) + 100.0*float64(score.Count100) + 300.0*float64(totalObjs-score.Count50-score.Count100)) / (300.0 * float64(totalObjs)) * 100.0
		go osutools.PPCalc(beatmap, accCalcNoMiss, "", "", scoreMods, ppValues)
		pp = "**" + strconv.FormatFloat(score.PP, 'f', 2, 64) + "pp**/" + <-ppValues + "pp "
	}
	acc := "** " + strconv.FormatFloat(accCalc, 'f', 2, 64) + "%** "
	hits := "**Hits:** [" + strconv.Itoa(score.Count300) + "/" + strconv.Itoa(score.Count100) + "/" + strconv.Itoa(score.Count50) + "/" + strconv.Itoa(score.CountMiss) + "]"

	g, _ := s.Guild(config.Conf.Server)
	tools.ErrRead(err)
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
			URL:     "https://osu.ppy.sh/users/" + strconv.Itoa(user.UserID),
			Name:    user.Username,
			IconURL: "https://a.ppy.sh/" + strconv.Itoa(user.UserID) + "?" + strconv.Itoa(rand.Int()) + ".jpeg",
		},
		Title: beatmap.Artist + " - " + beatmap.Title + " [" + beatmap.DiffName + "] by " + beatmap.Creator,
		URL:   "https://osu.ppy.sh/beatmaps/" + strconv.Itoa(beatmap.BeatmapID),
		Description: sr + length + bpm + "\n" +
			mapStats + "\n" +
			mapObjs + "\n\n" +
			scorePrint + mods + combo + acc + scoreRank + "\n" +
			mapCompletion + "\n" +
			pp + hits + "\n\n",
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://b.ppy.sh/thumb/" + strconv.Itoa(beatmap.BeatmapSetID) + "l.jpg",
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: time,
		},
	}
	if strings.ToLower(beatmap.Title) == "crab rave" {
		embed.Image = &discordgo.MessageEmbedImage{
			URL: "https://cdn.discordapp.com/emojis/510169818893385729.gif",
		}
	}
	s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content: warning,
		Embed:   embed,
	})
	return

	s.ChannelMessageSend(m.ChannelID, "Could not find any osu! account linked for "+m.Author.Mention()+" !")
	return
}

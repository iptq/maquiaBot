package handlers

import (
	"math/rand"
	"regexp"
	"strings"

	osuapi "../osu-api"
	osutools "../osu-functions"
	helpcommands "./help-commands"
	"github.com/bwmarrin/discordgo"
)

// HelpHandle lets you know the commands available
func HelpHandle(s *discordgo.Session, m *discordgo.MessageCreate, prefix string) {
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			URL:     "https://discordapp.com/oauth2/authorize?&client_id=" + s.State.User.ID + "&scope=bot&permissions=0",
			Name:    "Click here to invite MaquiaBot!",
			IconURL: s.State.User.AvatarURL("2048"),
		},
		Description: "**Most commands have other forms as well for convenience!**" + "\n\n" +
			"**Please do `" + prefix + "help <command>` for more information about the command!** \n" +
			"Help information format: `(cmd|names) <args> [optional args]`",
		Fields: []*discordgo.MessageEmbedField{
			&discordgo.MessageEmbedField{
				Name: "Admin commands:",
				Value: "`(prefix|maquiaprefix|newprefix)`, " +
					"`purge`, " +
					"`toggle`, " +
					"`(tr|track)`, " +
					"`(tt|trackt|ttoggle|tracktoggle)`",
			},
			&discordgo.MessageEmbedField{
				Name: "General commands:",
				Value: "`(adj|adjective|adjectives)`, " +
					"`(a|ava|avatar)`, " +
					"`(cc|cp|comparec|comparep|comparecock|comparepenis)`, " +
					"`(ch|choose)`, " +
					"`cheers`, " +
					"`crab`, " +
					"`decrypt`, " +
					"`(e|emoji|emote)`, " +
					"`encrypt`, " +
					"`face`, " +
					"`(idea|niceidea)`, " +
					"`info`, " +
					"`kanye`, " +
					"`(leven|levenshtein)`, " +
					"`(late|old|ancient)`, " +
					"`meme`, " +
					"`(noun|nouns)`, " +
					"`ocr`, " +
					"`(p|per|percent|percentage)`, " +
					"`parse`, " +
					"`(penis|cock)`, " +
					"`ping`, " +
					"`(q|quote)`, " +
					"`(qa|qadd|quotea|quoteadd)`, " +
					"`(qd|qr|qremove|qdelete|quoter|quoted|quoteremove|quotedelete)`, " +
					"`(qs|quotes)`, " +
					"`(rc|rp|rankc|rankp|rankcock|rankpenis)`, " +
					"`(remind|reminder)`, " +
					"`reminders`, " +
					"`(remindremove|rremove)`, " +
					"`(rinfo|roleinfo)`, " +
					"`roll`, " +
					"`(sinfo|serverinfo)`, " +
					"`(skill|skills)`, " +
					"`(stats|class)`, " +
					"`(twitter|twitterdl)`, " +
					"`(vibe|vibec|vibecheck)`",
			},
			&discordgo.MessageEmbedField{
				Name: "osu! commands:",
				Value: "`(bfarm|bottomfarm)`, " +
					"`bpm`, " +
					"`(c|compare)`, " +
					"`farm`, " +
					"`(l|leader|leaderboard)`, " +
					"`(link|set)`, " +
					"`(m|map)`, " +
					"`(osu|profile)`, " +
					"`osudetail`, " +
					"`osutop`, " +
					"`ppadd`, " +
					"`(r|rs|recent)`, " +
					"`(rb|recentb|recentbest)`, " +
					"`(t|top)`, " +
					"`(tfarm|topfarm)`, " +
					"`(ti|tinfo|tracking|trackinfo)`",
			},
			&discordgo.MessageEmbedField{
				Name: "Pokemon commands:",
				Value: "`(b|berry)`, " +
					"`pokemon`",
			},
		},
		Color: osutools.ModeColour(osuapi.ModeOsu),
	}

	argRegex, _ := regexp.Compile(`help\s+(.+)`)
	if argRegex.MatchString(m.Content) {
		arg := argRegex.FindStringSubmatch(m.Content)[1]
		args := strings.Split(arg, " ")
		if (args[0] == "pokemon" || args[0] == "osu") && len(args) > 1 {
			arg = args[1]
		}
		switch arg {
		// Admin commands
		case "prefix", "maquiaprefix", "newprefix":
			embed = helpcommands.Prefix(embed)
		case "purge":
			embed = helpcommands.Purge(embed)
		case "toggle":
			embed = helpcommands.Toggle(embed)
		case "tr", "track":
			embed = helpcommands.Track(embed)
		case "tt", "trackt", "ttoggle", "tracktoggle":
			embed = helpcommands.TrackToggle(embed)

		// General commands
		case "adj", "adjective", "adjectives":
			embed = helpcommands.Adjectives(embed)
		case "avatar", "ava", "a":
			embed = helpcommands.Avatar(embed)
		case "cc", "cp", "comparec", "comparep", "comparecock", "comparepenis":
			embed = helpcommands.PenisCompare(embed)
		case "ch", "choose":
			embed = helpcommands.Choose(embed)
		case "cheers":
			embed = helpcommands.Cheers(embed)
		case "crab":
			embed = helpcommands.Crab(embed)
		case "decrypt":
			embed = helpcommands.Decrypt(embed)
		case "e", "emoji", "emote":
			embed = helpcommands.Emoji(embed)
		case "encrypt":
			embed = helpcommands.Encrypt(embed)
		case "face":
			embed = helpcommands.Face(embed)
		case "idea", "niceidea":
			embed = helpcommands.NiceIdea(embed)
		case "info":
			embed = helpcommands.Info(embed)
		case "kanye":
			embed = helpcommands.Kanye(embed)
		case "leven", "levenshtein":
			embed = helpcommands.Levenshtein(embed)
		case "late", "old", "ancient":
			embed = helpcommands.Late(embed)
		case "meme":
			embed = helpcommands.Meme(embed)
		case "noun", "nouns":
			embed = helpcommands.Nouns(embed)
		case "ocr":
			embed = helpcommands.OCR(embed)
		case "p", "per", "percent", "percentage":
			embed = helpcommands.Percentage(embed)
		case "parse":
			embed = helpcommands.Parse(embed)
		case "penis", "cock":
			embed = helpcommands.Penis(embed)
		case "ping":
			embed = helpcommands.Ping(embed)
		case "q", "quote":
			embed = helpcommands.Quote(embed)
		case "qa", "qadd", "quotea", "quoteadd":
			embed = helpcommands.QuoteAdd(embed)
		case "qd", "qr", "qremove", "qdelete", "quoter", "quoted", "quoteremove", "quotedelete":
			embed = helpcommands.QuoteRemove(embed)
		case "qs", "quotes":
			embed = helpcommands.Quotes(embed)
		case "rc", "rp", "rankc", "rankp", "rankcock", "rankpenis":
			embed = helpcommands.PenisRank(embed)
		case "remind", "reminder":
			embed = helpcommands.Remind(embed)
		case "reminders":
			embed = helpcommands.Reminders(embed)
		case "remindremove", "rremove":
			embed = helpcommands.RemindRemove(embed)
		case "rinfo", "roleinfo":
			embed = helpcommands.RoleInfo(embed)
		case "roll":
			embed = helpcommands.Roll(embed)
		case "sinfo", "serverinfo":
			embed = helpcommands.ServerInfo(embed)
		case "skill", "skills":
			embed = helpcommands.Skills(embed)
		case "stats", "class":
			embed = helpcommands.Stats(embed)
		case "twitter", "twitterdl":
			embed = helpcommands.Twitter(embed)
		case "vibe", "vibec", "vibecheck":
			embed = helpcommands.Vibe(embed)

		// osu! commands
		case "bfarm", "bottomfarm":
			embed = helpcommands.BottomFarm(embed)
		case "bpm":
			embed = helpcommands.BPM(embed)
		case "c", "compare":
			embed = helpcommands.Compare(embed)
		case "farm":
			embed = helpcommands.Farm(embed)
		case "l", "leader", "leaderboard":
			embed = helpcommands.Leaderboard(embed)
		case "link", "set":
			embed = helpcommands.Link(embed)
		case "m", "map":
			embed = helpcommands.Map(embed)
		case "osu", "profile":
			embed = helpcommands.Profile(embed)
		case "osudetail":
			embed = helpcommands.ProfileDetail(embed)
		case "osutop":
			embed = helpcommands.ProfileTop(embed)
		case "ppadd":
			embed = helpcommands.PPAdd(embed)
		case "r", "rs", "recent":
			embed = helpcommands.Recent(embed)
		case "rb", "recentb", "recentbest":
			embed = helpcommands.RecentBest(embed)
		case "t", "top":
			embed = helpcommands.Top(embed)
		case "tfarm", "topfarm":
			embed = helpcommands.TopFarm(embed)
		case "ti", "tinfo", "tracking", "trackinfo":
			embed = helpcommands.TrackInfo(embed)

		// Pokemon commands
		case "b", "berry":
			embed = helpcommands.Berry(embed)
		case "pokemon":
			embed = helpcommands.Pokemon(embed)
		}
	}

	if !strings.HasPrefix(embed.Description, "**Most") && embed.Fields[0].Name == "Admin commands:" {
		embed.Fields = []*discordgo.MessageEmbedField{}
	}

	switch rand.Intn(11) {
	case 0:
		embed.Image = &discordgo.MessageEmbedImage{
			URL: "https://cdn.discordapp.com/attachments/555493588465877012/555994312760885248/epicAnimeScene.gif",
		}
	case 1:
		embed.Image = &discordgo.MessageEmbedImage{
			URL: "https://cdn.discordapp.com/attachments/555493588465877012/555996915884490752/epicAnimeGifTWO.gif",
		}
	case 2:
		embed.Image = &discordgo.MessageEmbedImage{
			URL: "https://cdn.discordapp.com/attachments/555493588465877012/556000178406948875/epicAnimeGif5.gif",
		}
	case 3, 4:
		embed.Image = &discordgo.MessageEmbedImage{
			URL: "https://cdn.discordapp.com/attachments/555493588465877012/555998627940532237/tumblr_phjkel3lgn1xlyyvto1_1280.png",
		}
	case 5, 6:
		embed.Image = &discordgo.MessageEmbedImage{
			URL: "https://cdn.discordapp.com/attachments/555493588465877012/555998640142024704/tumblr_phjkel3lgn1xlyyvto2_1280.png",
		}
	case 7, 8:
		embed.Image = &discordgo.MessageEmbedImage{
			URL: "https://cdn.discordapp.com/attachments/555493588465877012/555998669761937418/tumblr_phjkel3lgn1xlyyvto3_1280.png",
		}
	case 9, 10:
		embed.Image = &discordgo.MessageEmbedImage{
			URL: "https://cdn.discordapp.com/attachments/555493588465877012/555998681375965194/tumblr_phjkel3lgn1xlyyvto5_1280.png",
		}
	}
	s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
		Content: "All commands in PM will use the bot's default prefix `$` instead!",
		Embed:   embed,
	})
}

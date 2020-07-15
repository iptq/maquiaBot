package gencommands

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	config "maquiaBot/config"
	structs "maquiaBot/structs"
	tools "maquiaBot/tools"

	"github.com/bwmarrin/discordgo"
)

// ReminderTimers is the list of all reminders running
var ReminderTimers []structs.ReminderTimer

// Remind reminds the person after an x amount of specified time
func Remind(s *discordgo.Session, m *discordgo.MessageCreate) {
}

// RunReminder runs the reminder
func RunReminder(s *discordgo.Session, reminderTimer structs.ReminderTimer) {
	if time.Now().Before(reminderTimer.Reminder.Target) {
		<-reminderTimer.Timer.C
	}
	for i, rmndr := range ReminderTimers {
		if rmndr.Reminder.ID == reminderTimer.Reminder.ID {
			if rmndr.Reminder.Active {
				go ReminderMessage(s, reminderTimer)
			}
			ReminderTimers[i] = ReminderTimers[len(ReminderTimers)-1]
			ReminderTimers = ReminderTimers[:len(ReminderTimers)-1]
			break
		}
	}

	// Remove reminder
	reminders := []structs.Reminder{}
	_, err := os.Stat("./data/reminders.json")
	if err == nil {
		f, err := ioutil.ReadFile("./data/reminders.json")
		tools.ErrRead(s, err)
		_ = json.Unmarshal(f, &reminders)
	} else {
		tools.ErrRead(s, err)
	}
	for i, reminder := range reminders {
		if reminder.ID == reminderTimer.Reminder.ID {
			reminders[i] = reminders[len(reminders)-1]
			reminders = reminders[:len(reminders)-1]
			break
		}
	}

	// Save reminders
	jsonCache, err := json.Marshal(reminders)
	tools.ErrRead(s, err)

	err = ioutil.WriteFile("./data/reminders.json", jsonCache, 0644)
	tools.ErrRead(s, err)
}

// Reminders lists the person's reminders
func Reminders(s *discordgo.Session, m *discordgo.MessageCreate) {
	userTimers := []structs.Reminder{}
	all := false
	if strings.Contains(m.Content, "all") {
		if m.Author.ID != config.Conf.BotHoster.UserID {
			s.ChannelMessageSend(m.ChannelID, "YOU ARE NOT "+config.Conf.BotHoster.Username+".........")
			return
		}
		all = true
		for _, reminder := range ReminderTimers {
			if reminder.Reminder.Active {
				userTimers = append(userTimers, reminder.Reminder)
			}
		}
	} else {
		for _, reminder := range ReminderTimers {
			if reminder.Reminder.User.ID == m.Author.ID && reminder.Reminder.Active {
				userTimers = append(userTimers, reminder.Reminder)
			}
		}
	}

	if len(userTimers) == 0 {
		s.ChannelMessageSend(m.ChannelID, "You have no pending reminders!")
		return
	}

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    m.Author.String(),
			IconURL: m.Author.AvatarURL("2048"),
		},
		Description: "Please use `rremove <ID>` or `remindremove <ID>` to remove a reminder",
	}
	if all {
		for _, reminder := range userTimers {
			info := reminder.Info
			if info == "" {
				info = "N/A"
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   strconv.FormatInt(reminder.ID, 10),
				Value:  "Reminder: " + info + "\nRemind time: " + reminder.Target.Format(time.RFC822) + "\nUser: " + reminder.User.String(),
				Inline: true,
			})
		}
	} else {
		for _, reminder := range userTimers {
			info := reminder.Info
			if info == "" {
				info = "N/A"
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   strconv.FormatInt(reminder.ID, 10),
				Value:  "Reminder: " + info + "\nRemind time: " + reminder.Target.Format(time.RFC822),
				Inline: true,
			})
		}
	}
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

// RemoveReminder removes a reminder (kind of)
func RemoveReminder(s *discordgo.Session, m *discordgo.MessageCreate) {
	remindRegex, _ := regexp.Compile(`(?i)r(emind)?remove\s+(\d+|all)`)
	if !remindRegex.MatchString(m.Content) {
		s.ChannelMessageSend(m.ChannelID, "Please give a reminder's snowflake ID to remove! You can see all of your reminds with `reminders`. If you want to remove all reminders, please state `remindremove all`")
		return
	}

	// Get reminders
	reminders := []structs.Reminder{}
	_, err := os.Stat("./data/reminders.json")
	if err == nil {
		f, err := ioutil.ReadFile("./data/reminders.json")
		tools.ErrRead(s, err)
		_ = json.Unmarshal(f, &reminders)
	} else {
		tools.ErrRead(s, err)
	}

	// Mark Active as false for the reminder in both slices
	reminderID := remindRegex.FindStringSubmatch(m.Content)[2]
	if reminderID == "all" {
		for i, reminder := range ReminderTimers {
			if reminder.Reminder.User.ID == m.Author.ID {
				ReminderTimers[i].Reminder.Active = false
			}
		}
		for i, reminder := range reminders {
			if reminder.User.ID == m.Author.ID {
				reminders[i].Active = false
			}
		}
		s.ChannelMessageSend(m.ChannelID, "Removed reminders!")
	} else {
		reminderIDint, err := strconv.ParseInt(reminderID, 10, 64)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error parsing ID.")
			return
		}
		for i, reminder := range ReminderTimers {
			if reminder.Reminder.ID == reminderIDint {
				ReminderTimers[i].Reminder.Active = false
				break
			}
		}
		for i, reminder := range reminders {
			if reminder.ID == reminderIDint {
				reminders[i].Active = false
				break
			}
		}
		s.ChannelMessageSend(m.ChannelID, "Removed reminder!")
	}

	// Save reminders
	jsonCache, err := json.Marshal(reminders)
	tools.ErrRead(s, err)

	err = ioutil.WriteFile("./data/reminders.json", jsonCache, 0644)
	tools.ErrRead(s, err)
}

// ReminderMessage will send the user their reminder
func ReminderMessage(s *discordgo.Session, reminderTimer structs.ReminderTimer) {
	linkRegex, _ := regexp.Compile(`(?i)https?:\/\/\S+`)
	dm, _ := s.UserChannelCreate(reminderTimer.Reminder.User.ID)
	if reminderTimer.Reminder.Info == "" {
		s.ChannelMessageSend(dm.ID, "Reminder!")
	} else if linkRegex.MatchString(reminderTimer.Reminder.Info) {
		response, err := http.Get(linkRegex.FindStringSubmatch(reminderTimer.Reminder.Info)[0])
		if err != nil {
			s.ChannelMessageSend(dm.ID, "Reminder about `"+reminderTimer.Reminder.Info+"`!")
			return
		}
		img, _, err := image.Decode(response.Body)
		if err != nil {
			s.ChannelMessageSend(dm.ID, "Reminder about `"+reminderTimer.Reminder.Info+"`!")
			return
		}
		imgBytes := new(bytes.Buffer)
		err = png.Encode(imgBytes, img)
		if err != nil {
			s.ChannelMessageSend(dm.ID, "Reminder about `"+reminderTimer.Reminder.Info+"`!")
		}
		_, err = s.ChannelMessageSendComplex(dm.ID, &discordgo.MessageSend{
			Content: "Reminder about this",
			Files: []*discordgo.File{
				{
					Name:   "image.png",
					Reader: imgBytes,
				},
			},
		})
		if err != nil {
			s.ChannelMessageSend(dm.ID, "Reminder about `"+reminderTimer.Reminder.Info+"`!")
		}
		response.Body.Close()
		return
	} else {
		s.ChannelMessageSend(dm.ID, "Reminder about `"+reminderTimer.Reminder.Info+"`!")
	}
}

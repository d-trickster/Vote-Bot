package bot

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
	"vote/tgclient"
)

const (
	cmdHelp  = "help"
	cmdStart = "start"

	cmdAdd        = "add"
	cmdStatus     = "status"
	cmdStatusFull = "status_full"
	cmdVote       = "vote"

	// admin commands
	cmdMonitor = "monitor"
	cmdReboot  = "reboot"
	cmdRemove  = "remove"
	cmdReset   = "reset"
)

func (b *Bot) processCommand(update *tgclient.Update) {
	var ent tgclient.Enitiy
	for _, e := range update.Message.Entities {
		if e.Type == tgclient.EntityBotCommand {
			ent = e
			break
		}
	}
	if ent.Len == 0 {
		return
	}

	sep := ent.Offset + ent.Len
	cmd := update.Message.Text[ent.Offset+1 : sep]
	for i := range cmd {
		if cmd[i] == '@' {
			cmd = cmd[:i]
			break
		}
	}

	switch cmd {

	case cmdHelp:
		b.help(&update.Message)
	case cmdStart:
		b.register(&update.Message)

	case cmdAdd:
		b.addFilm(&update.Message, strings.TrimSpace(update.Message.Text[sep:]))
		b.monitorCh <- struct{}{}
	case cmdStatus:
		b.status(&update.Message)
	case cmdStatusFull:
		b.statusFull(&update.Message)
	case cmdVote:
		b.vote(&update.Message)

	case cmdReboot:
		b.reboot(&update.Message)
	case cmdMonitor:
		b.monitor(&update.Message)
	case cmdRemove:
		b.remove(&update.Message, strings.TrimSpace(update.Message.Text[sep:]))
		b.monitorCh <- struct{}{}
	case cmdReset:
		b.reset(&update.Message)
		b.monitorCh <- struct{}{}
	}
}

func (b *Bot) help(msg *tgclient.Message) {
	var text string
	if b.isAdmin(msg.From.Id) {
		text = msgHelpAdmin
	} else {
		text = msgHelp
	}

	if err := b.client.Answer(msg, text); err != nil {
		slog.Error(err.Error())
	}
}

func (b *Bot) addFilm(msg *tgclient.Message, film string) {
	if film == "" {
		if err := b.client.Answer(
			msg,
			"Invalid film name 🤡\n<span class=\"tg-spoiler\">Usage: /add Зелёный слоник 2</span>",
		); err != nil {
			slog.Error(err.Error())
		}
		return
	}
	if err := b.storage.AddFilm(msg.From.Id, film); err != nil {
		slog.Error("Faield to handle addFilm: " + err.Error())
	} else {
		if err := b.client.Answer(msg, fmt.Sprintf("\"%s\" добавлен в список 📋✍️", film)); err != nil {
			slog.Error(err.Error())
		}
	}
}

func (b *Bot) register(msg *tgclient.Message) {
	added, err := b.storage.Register(msg.From.Id, msg.From.Name, msg.From.Username)
	if err != nil {
		slog.Error("Failed to register user: " + err.Error())
	}
	if added {
		if err := b.client.Answer(msg, "<b>Welcome to the club, buddy</b> 🍑👋"); err != nil {
			slog.Error(err.Error())
		}
	} else {
		if err := b.client.Answer(msg, "Ты уже смешарик..."); err != nil {
			slog.Error(err.Error())
		}
	}
}

func (b *Bot) status(msg *tgclient.Message) {
	stats := b.storage.Status()

	var vote int
	if msg.Chat.Type == tgclient.ChatTypePrivate {
		vote = b.storage.GetVote(msg.From.Id)
	}
	text := statusText(stats, vote)

	if err := b.client.Answer(msg, text); err != nil {
		slog.Error(fmt.Sprintf("failed to handle status requst: %s", err.Error()))
	}
}

func (b *Bot) statusFull(msg *tgclient.Message) {
	stats := b.storage.StatusFull()
	if len(stats) == 0 {
		if err := b.client.Answer(msg, "Фильмов пока нет 💀"); err != nil {
			slog.Error(err.Error())
		}
		return
	}

	builder := strings.Builder{}
	for i := range stats {
		builder.WriteString(fmt.Sprintf(
			"🔸 <b>%s</b> by <a href=\"https://t.me/%s\">%s</a> - %d:\n",
			stats[i].Name, stats[i].AddedBy.Username, stats[i].AddedBy.Name, stats[i].Votes,
		))
		for j := range stats[i].Voters {
			builder.WriteString(fmt.Sprintf(
				"<a href=\"https://t.me/%s\">%s</a>\n",
				stats[i].Voters[j].Username, stats[i].Voters[j].Name,
			))
		}
	}

	if err := b.client.Answer(msg, builder.String()); err != nil {
		slog.Error(fmt.Sprintf("failed to handle status requst: %s", err.Error()))
	}
}

func (b *Bot) vote(msg *tgclient.Message) {
	stats := b.storage.Status()

	n := len(stats)
	keyboard := tgclient.InlineKeyboardMarkup{
		Keyboard: make([][]tgclient.InlineKeyboardButton, n+1),
	}
	for i := range n {
		keyboard.Keyboard[i] = []tgclient.InlineKeyboardButton{{
			Text: stats[i].Name,
			Data: fmt.Sprintf("%s%d", prefVote, stats[i].Id),
		}}
	}
	keyboard.Keyboard[n] = []tgclient.InlineKeyboardButton{{
		Text: "❌",
		Data: prefVote + "0",
	}}

	if err := b.client.SendInlineKeyboard(msg.From.Id, "🤔🤔🤔🤔", keyboard); err != nil {
		slog.Error("Failed to send voting message: " + err.Error())
	}
}

// admin command
func (b *Bot) remove(msg *tgclient.Message, film string) {
	if !b.isAdmin(msg.From.Id) {
		if err := b.client.Answer(msg, "Кыш 😡"); err != nil {
			slog.Error(err.Error())
		}
		return
	}

	found, err := b.storage.RemoveFilm(film)
	if err != nil {
		slog.Error("failed to remove film: " + err.Error())
		if err := b.client.Answer(msg, err.Error()); err != nil {
			slog.Error(err.Error())
		}
	}

	if found {
		if err := b.client.Answer(msg, film+" removed"); err != nil {
			slog.Error(err.Error())
		}
	} else {
		if err := b.client.Answer(msg, film+" wasn't found"); err != nil {
			slog.Error(err.Error())
		}
	}
}

// admin command
func (b *Bot) reset(msg *tgclient.Message) {
	if !b.isAdmin(msg.From.Id) {
		if err := b.client.Answer(msg, "Кыш 😡"); err != nil {
			slog.Error(err.Error())
		}
		return
	}

	b.storage.ResetVotes()
	if err := b.client.Answer(msg, "Голоса сброшены"); err != nil {
		slog.Error(err.Error())
	}
}

// admin command
func (b *Bot) monitor(msg *tgclient.Message) {
	if !b.isAdmin(msg.From.Id) {
		if err := b.client.Answer(msg, "Кыш 😡"); err != nil {
			slog.Error(err.Error())
		}
		return
	}

	var text string
	stats := b.storage.Status()
	if len(stats) == 0 {
		text = "Фильмов пока нет 💀"
	} else {
		first, second := getPositions(stats)
		builder := strings.Builder{}
		for i := range stats {
			builder.WriteString(fmt.Sprintf(
				"%s<b>%s</b>: %d\n",
				getEmoj(i, first, second), stats[i].Name, stats[i].Votes,
			))
		}
		text = builder.String()
	}

	m, err := b.client.AnswerWithResult(msg, text)
	if err != nil {
		slog.Error(err.Error())
	}

	b.storage.SetMonitor(m.Chat.Id, m.Id)
}

// admin command
func (b *Bot) reboot(msg *tgclient.Message) {
	if !b.isAdmin(msg.From.Id) {
		if err := b.client.Answer(msg, "Кыш 😡"); err != nil {
			slog.Error(err.Error())
		}
		return
	}
	if time.Since(b.startTime) > time.Second*10 {
		b.Stop()
	}
}

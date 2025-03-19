package bot

import (
	"log/slog"
	"math/rand"
	"strconv"
	"strings"
	"vote/tgclient"
)

const (
	prefSize = 4
	prefVote = "film"
)

func (b *Bot) processCallback(update *tgclient.Update) {
	emptyKeyboard := tgclient.InlineKeyboardMarkup{Keyboard: [][]tgclient.InlineKeyboardButton{}}

	if strings.HasPrefix(update.Callback.Data, prefVote) {
		id, err := strconv.ParseInt(update.Callback.Data[prefSize:], 10, 64)
		if err != nil {
			slog.Error("Failed to parse callback data: " + err.Error())
		}

		ok, err := b.storage.Vote(update.Callback.From.Id, int(id))
		if err != nil {
			slog.Error("Failed to process callback: " + err.Error())
		}

		if ok {
			err = b.client.EditMessage(
				update.Callback.From.Id,
				update.Callback.Message.Id,
				"Отличный выбор "+randEmoji(),
				emptyKeyboard,
			)
		} else {
			err = b.client.EditMessage(
				update.Callback.From.Id,
				update.Message.Id,
				"Что-то пошло не так",
				emptyKeyboard,
			)
		}
		if err != nil {
			slog.Error("Failed to send message after vote: " + err.Error())
		}
		b.monitorCh <- struct{}{}
	}
}

var emojis = []rune("🫡🤯💩🤡👍👎😡🤓🌚🔥")

func randEmoji() string {
	i := rand.Intn(len(emojis))
	return string(emojis[i])
}

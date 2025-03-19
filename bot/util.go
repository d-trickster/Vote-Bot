package bot

import (
	"fmt"
	"slices"
	"strings"
	"vote/storage"
)

const (
	msgHelp = `<b>Я бот.</b>

/status - посмотреть список фильмов и голосов
/vote - проголосовать за фильм (в лс)
/add Борат 2 - добавить фильм в список
/status_full - посмотреть голоса

/start - начало работы (должна быть отправлена хотябы раз!)
/help - помощь`
	msgHelpAdmin = msgHelp + `

<b>Админские команды</b> 😈:
/remove Борат 2 - удалить фильм из списка
/monitor - обновляющийсяя в реальном времени status (работает только последнее сообщение)
/reset - сбрасывает ВСЕ голоса`
	msgAddNoFilm = "Invalid film name 🤡\n<span class=\"tg-spoiler\">Usage: /add Зелёный слоник 2</span>"
	msgAddedTmpl = "\"%s\" добавлен в список 📋✍️"
)

func (b *Bot) isAdmin(userID int64) bool {
	return slices.Contains(b.admins, userID)
}

func (b *Bot) loadAdmins() error {
	admins, err := b.client.ChatAdmins(b.mainChatId)
	if err != nil {
		return err
	}

	uniq := map[int64]struct{}{}

	for _, id := range b.admins {
		uniq[id] = struct{}{}
	}
	for _, adm := range admins {
		uniq[adm.User.Id] = struct{}{}
	}

	res := make([]int64, 0, len(uniq))
	for id := range uniq {
		res = append(res, id)
	}
	b.admins = res

	return nil
}

func statusText(stats []storage.FilmStat, vote int) string {
	if len(stats) == 0 {
		return "Фильмов пока нет 💀"
	}

	first, second := getPositions(stats)
	builder := strings.Builder{}
	for i := range stats {
		emoj := ""
		if stats[i].Id == vote {
			emoj = " 💋"
		}
		builder.WriteString(fmt.Sprintf(
			"%s<b>%s</b>: %d%s\n",
			getEmoj(i, first, second), stats[i].Name, stats[i].Votes, emoj,
		))
	}

	return builder.String()
}

func getPositions(stats []storage.FilmStat) (first, second int) {
	max1, max2 := stats[0].Votes, 0
	for i := range stats {
		cur := stats[i].Votes
		if cur == max1 {
			first++
		} else if cur == max2 {
			second++
		} else if cur > max2 {
			if max2 == 0 {
				max2 = cur
				second++
			} else {
				break
			}
		}
	}
	return
}

func getEmoj(i int, first int, second int) string {
	if first > 3 {
		return ""
	}
	if i < first {
		if first == 1 {
			return "🏆 "
		}
		return "🥇 "
	}
	if i < first+second {
		if second > 3 {
			return ""
		}
		return "🥈 "
	}
	return ""
}

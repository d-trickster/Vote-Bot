package bot

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
	"vote/config"
	"vote/storage"
	"vote/tgclient"
)

type Bot struct {
	client  *tgclient.Client
	storage *storage.Storage

	fetchInterval time.Duration
	limit         int
	offset        int

	admins     []int64
	mainChatId int64
	monitors   []int64
	monitorCh  chan struct{}
	startTime  time.Time
	stopCh     chan struct{}
	doneCh     chan struct{}
}

func New(cfg *config.Config, token string) (*Bot, error) {
	if token == "" {
		return nil, fmt.Errorf("no token provided")
	}
	st, err := storage.New(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}
	return &Bot{
		client:        tgclient.NewClient(token),
		storage:       st,
		admins:        cfg.Admins,
		mainChatId:    cfg.MainChatId,
		fetchInterval: cfg.FetchInterval,
		limit:         cfg.Limit,
		offset:        cfg.Offset,
		monitorCh:     make(chan struct{}),
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}, nil
}

func (b *Bot) Start() {
	slog.Info("Bot is starting...")

	b.setCommands()

	b.startTime = time.Now()
	go b.startFetching()

	slog.Info("Bot is running")
}

func (b *Bot) Stop() {
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("bot.Stop() recovered")
		}
	}()
	close(b.stopCh)
}

func (b *Bot) Wait() <-chan struct{} {
	waitCh := make(chan struct{})

	go func() {
		<-b.doneCh
		close(waitCh)
	}()

	return waitCh
}

func (b *Bot) startFetching() {
	fetchTicker := time.NewTicker(b.fetchInterval)
	defer fetchTicker.Stop()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Debug(fmt.Sprintf("monitor: %v", b.storage.GetMonitor()))
		b.updateMonitors()
		slog.Debug("Monitor stopped")
	}()

	for {
		select {

		case <-fetchTicker.C:
			updates, err := b.client.Updates(b.limit, b.offset)
			if err != nil {
				slog.Error(fmt.Sprintf("error getting updates: %s", err.Error()))
				continue
			}
			if len(updates) == 0 {
				continue
			}
			slog.Debug(fmt.Sprintf("%v", updates))
			slog.Info(fmt.Sprintf("fetched %d updates", len(updates)))

			for i := range updates {
				wg.Add(1)
				go func() {
					defer wg.Done()
					b.process(updates[i])
				}()
			}

			b.offset = updates[len(updates)-1].Id + 1

		case <-b.stopCh:
			slog.Info("Waiting for processing to finish")
			wg.Wait()
			slog.Info("Processing finished")
			close(b.doneCh)
			return
		}
	}
}

func (b *Bot) process(update tgclient.Update) {
	slog.Info(
		"Processing update",
		"from", fmt.Sprintf("%s (@%s)", update.Message.From.Name, update.Message.From.Username),
		"text", update.Message.Text,
		"date", time.Unix(update.Message.Date, 0).Format("2006-01-02 15:04:05"),
	)

	if update.Callback.Data != "" {
		b.processCallback(&update)
	} else {
		b.processCommand(&update)
	}
}

func (b *Bot) updateMonitors() {
	slog.Debug("Monitoring...")
	defer slog.Debug("Monitoring stopped")

	for {
		select {
		case <-b.monitorCh:
			slog.Debug("Updating monitor!")
			mon := b.storage.GetMonitor()
			text := statusText(b.storage.Status(), 0)
			if err := b.client.EditMessage(
				mon.ChatId, mon.MsgId,
				text,
				tgclient.InlineKeyboardMarkup{Keyboard: [][]tgclient.InlineKeyboardButton{}},
			); err != nil {
				slog.Error("Failed to update monitor: " + err.Error())
			}

		case <-b.stopCh:
			slog.Debug("updateMonitors stopped")
			return
		}
	}
}

func (b *Bot) setCommands() {
	if err := b.loadAdmins(); err != nil {
		slog.Error("Faield to load chat admins: " + err.Error())
	}

	if err := b.client.SetCommandsPrivate([][]string{
		{cmdStatus, "Посмотреть список фильмов"},
		{cmdVote, "Голосовать за фильм"},
		{cmdAdd, "Добавть фильм в список"},
		{cmdStatusFull, "Список фильмов с голосами"},
		{cmdHelp, "Помощь"},
	}); err != nil {
		slog.Error("Failed to set private commands: " + err.Error())
	}
	// for _, id := range b.admins {
	// 	if err := b.client.SetCommandChat([][]string{
	// 		{cmdStatus, "Посмотреть список фильмов"},
	// 		{cmdVote, "Голосовать за фильм"},
	// 		{cmdAdd, "Добавть фильм в список"},
	// 		{cmdStatusFull, "Список фильмов с голосами"},
	// 		{cmdHelp, "Помощь"},
	// 		{cmdRemove, "😈 Удалить фильм из списка"},
	// 		{cmdReset, "😈 Сбросить ВСЕ голоса"},
	// 		{cmdMonitor, "😈 Сообщение /status с автообновлением"},
	// 	}, id); err != nil {
	// 		slog.Error("failed to set private commands for admons: " + err.Error())
	// 	}
	// }
	if err := b.client.SetCommandsGroup([][]string{
		{cmdStatus, "Посмотреть список фильмов"},
		{cmdAdd, "Добавть фильм в список"},
		{cmdStatusFull, "Список фильмов с голосами"},
		{cmdHelp, "Помощь"},
	}); err != nil {
		slog.Error("failed to set group commands: " + err.Error())
	}
	if err := b.client.SetCommandsGroupAdmin([][]string{
		{cmdStatus, "Посмотреть список фильмов"},
		{cmdAdd, "Добавть фильм в список"},
		{cmdStatusFull, "Список фильмов с голосами"},
		{cmdHelp, "Помощь"},
		{cmdRemove, "😈 Удалить фильм из списка"},
		{cmdReset, "😈 Сбросить ВСЕ голоса"},
		{cmdMonitor, "😈 Сообщение /status с автообновлением"},
	}); err != nil {
		slog.Error("failed to set group admin commands: " + err.Error())
	}
}

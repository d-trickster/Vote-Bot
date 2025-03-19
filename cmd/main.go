package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"vote/bot"
	"vote/config"
	"vote/logging"
)

func main() {
	cfg := config.MustLoad()

	logFile := logging.SetupLogging(cfg.Env, cfg.LogPath)
	if logFile != nil {
		defer logFile.Close()
	}
	slog.Info("Logging started")

	tgbot, err := bot.New(cfg, os.Getenv("BOT_TOKEN"))
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to create bot: %s", err.Error()))
		return
	}

	tgbot.Start()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		slog.Warn("Shutdown signal received")
		tgbot.Stop()
		<-tgbot.Wait()
	case <-tgbot.Wait():
	}

	slog.Warn("Bot has been shut down")
}

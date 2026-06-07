package harness

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

type telegramBot struct {
	token       string
	allowedChat string
	packsRoot   string
	defaultPack string
	repoRoot    string
	client      *http.Client
}

type telegramUpdate struct {
	UpdateID int `json:"update_id"`
	Message  struct {
		MessageID int    `json:"message_id"`
		Date      int64  `json:"date"`
		Text      string `json:"text"`
		Chat      struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		From struct {
			ID       int64  `json:"id"`
			Username string `json:"username"`
		} `json:"from"`
	} `json:"message"`
}

func runTelegram(args []string) int {
	fs := flag.NewFlagSet("telegram", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	token := fs.String("token", os.Getenv("TELEGRAM_BOT_TOKEN"), "telegram bot token")
	allowedChat := fs.String("allowed-chat-id", os.Getenv("TELEGRAM_ALLOWED_CHAT_ID"), "allowed chat id")
	packsRoot := fs.String("packs-root", envDefault("WORLD_HARNESS_PACKS_ROOT", "packs"), "packs root")
	defaultPack := fs.String("default-pack", envDefault("WORLD_HARNESS_DEFAULT_PACK", "lumen-federation"), "default pack")
	repoRoot := fs.String("repo-root", envDefault("WORLD_HARNESS_REPO_ROOT", "."), "repo root")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *token == "" {
		fmt.Fprintln(os.Stderr, "TELEGRAM_BOT_TOKEN is required")
		return 2
	}
	bot := &telegramBot{
		token:       *token,
		allowedChat: *allowedChat,
		packsRoot:   *packsRoot,
		defaultPack: *defaultPack,
		repoRoot:    *repoRoot,
		client:      &http.Client{Timeout: 90 * time.Second},
	}
	return bot.loop()
}

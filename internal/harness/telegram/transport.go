package telegram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"time"
)

func (b *telegramBot) loop() int {
	offset := 0
	for {
		updates, err := b.getUpdates(offset)
		if err != nil {
			fmt.Fprintln(os.Stderr, "telegram getUpdates:", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, update := range updates {
			offset = update.UpdateID + 1
			if update.Message.Text == "" {
				continue
			}
			chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
			if b.allowedChat != "" && chatID != b.allowedChat {
				_ = b.sendMessage(chatID, "이 봇은 허용된 채팅에서만 world-harness 명령을 처리합니다.")
				continue
			}
			reply := b.handle(update)
			if reply != "" {
				_ = b.sendMessage(chatID, reply)
			}
		}
	}
}

func (b *telegramBot) getUpdates(offset int) ([]telegramUpdate, error) {
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?timeout=60&offset=%d", url.PathEscape(b.token), offset)
	resp, err := b.client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		OK     bool             `json:"ok"`
		Result []telegramUpdate `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if !parsed.OK {
		return nil, fmt.Errorf("telegram returned not ok: %s", string(body))
	}
	return parsed.Result, nil
}

func (b *telegramBot) sendMessage(chatID, text string) error {
	if len(text) > 3900 {
		text = text[:3900] + "\n..."
	}
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")
	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", url.PathEscape(b.token))
	resp, err := b.client.PostForm(endpoint, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

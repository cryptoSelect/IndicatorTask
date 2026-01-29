package notify

import (
	"IndicatorTask/config"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func SendTelegramMessage(MsgTopic, Msg string) bool {
	var topic string

	switch MsgTopic {
	case "5m":
		topic = config.Cfg.Notify.Topic5Minue
	case "15m":
		topic = config.Cfg.Notify.Topic15Minue
	case "30m":
		topic = config.Cfg.Notify.Topic30Minue
	case "1h":
		topic = config.Cfg.Notify.Topic1Hour
	case "4h":
		topic = config.Cfg.Notify.Topic4Hour
	case "1d":
		topic = config.Cfg.Notify.Topic1Day
	case "1w":
		topic = config.Cfg.Notify.Topic1Week
	case "1M":
		topic = config.Cfg.Notify.Topic1Month
	default:
		topic = config.Cfg.Notify.InformationTopic
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.Cfg.Notify.Token)
	body, _ := json.Marshal(map[string]string{
		"chat_id":           config.Cfg.Notify.Group,
		"message_thread_id": topic,
		"text":              Msg,
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("send telegram message error:", err.Error())
	}
	defer resp.Body.Close()
	return true
}

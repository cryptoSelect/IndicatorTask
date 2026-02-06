// Package notify 提供通知队列：计算任务将待通知消息入队，Worker 按订阅关系发送给对应用户
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"IndicatorTask/config"
	"IndicatorTask/utils/logger"

	"github.com/cryptoSelect/public/database"
)

const queueCap = 500

// NotifyJob 待发送的通知任务
type NotifyJob struct {
	Symbol  string
	Cycle   string
	Message string
}

var (
	queue   chan NotifyJob
	once    sync.Once
	started bool
)

func initQueue() {
	queue = make(chan NotifyJob, queueCap)
}

// Push 将通知任务放入队列（非阻塞，队列满则丢弃）
func Push(symbol, cycle, message string) {
	once.Do(initQueue)
	if !started {
		logger.Log.Warn("notify worker not started, job dropped", map[string]interface{}{"symbol": symbol, "cycle": cycle})
		return
	}
	select {
	case queue <- NotifyJob{Symbol: symbol, Cycle: cycle, Message: message}:
	default:
		logger.Log.Warn("notify queue full, job dropped", map[string]interface{}{"symbol": symbol, "cycle": cycle})
	}
}

// StartWorker 启动通知 Worker，从队列消费并按订阅关系发送给用户
func StartWorker(ctx context.Context) {
	once.Do(initQueue)
	started = true
	logger.Log.Info("notify worker started", nil)
	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("notify worker stopped", nil)
			return
		case job, ok := <-queue:
			if !ok {
				return
			}
			sendToSubscribers(job)
		}
	}
}

// sendToSubscribers 查询订阅该 symbol 的用户，向有 telegram_id 的用户发送消息
func sendToSubscribers(job NotifyJob) {
	if database.DB == nil {
		return
	}
	symbol := strings.TrimSpace(strings.ToUpper(job.Symbol))
	cycle := strings.TrimSpace(job.Cycle)
	if symbol == "" || cycle == "" || job.Message == "" {
		return
	}

	// 查询：订阅了该 symbol+cycle 且已绑定 telegram 的用户
	var rows []struct {
		TelegramID string `gorm:"column:telegram_id"`
	}
	err := database.DB.Raw(`
		SELECT u.telegram_id FROM user_info u
		JOIN subscription s ON s.user_id = u.id
		WHERE s.symbol = ? AND s.cycle = ? AND u.telegram_id IS NOT NULL AND TRIM(u.telegram_id) != ''
	`, symbol, cycle).Scan(&rows).Error
	if err != nil {
		logger.Log.Error("query subscribers failed", map[string]interface{}{"symbol": symbol, "err": err.Error()})
		return
	}
	telegramIDs := make([]string, 0, len(rows))
	for _, r := range rows {
		if t := strings.TrimSpace(r.TelegramID); t != "" {
			telegramIDs = append(telegramIDs, t)
		}
	}
	if len(telegramIDs) == 0 {
		return
	}

	token := strings.TrimSpace(config.Cfg.Notify.Token)
	if token == "" {
		logger.Log.Warn("telegram token not configured", nil)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	for _, chatID := range telegramIDs {
		chatID = strings.TrimSpace(chatID)
		if chatID == "" {
			continue
		}
		body, _ := json.Marshal(map[string]string{
			"chat_id": chatID,
			"text":    job.Message,
		})
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			logger.Log.Error("send telegram failed", map[string]interface{}{"chat_id": chatID, "err": err.Error()})
			continue
		}
		resp.Body.Close()
	}
}

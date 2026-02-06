package main

import (
	"IndicatorTask/binanceFapi"
	"IndicatorTask/calculate"
	"IndicatorTask/clean"
	"IndicatorTask/config"
	"IndicatorTask/utils/logger"
	"IndicatorTask/utils/notify"
	"context"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cryptoSelect/public/database"
	"github.com/cryptoSelect/public/models"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {

	config.Init()
	logger.Init(config.Cfg.Mode)
	db := config.Cfg.Database
	database.InitDB(db.Host, db.User, db.Password, db.DBName, db.Port)
	if err := database.AutoMigrate(&models.SymbolRecord{}, &models.UserInfo{}, &models.Subscription{}); err != nil {
		panic("failed to migrate database: " + err.Error())
	}
	clean.CleanNaNData()

	ctx, cancle := context.WithCancel(context.Background())
	defer cancle()

	// 捕获 Ctrl+C 信号
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		<-c
		logger.Log.Info("收到退出信号，应用关闭中")
		cancle()
	}()
	if len(config.Cfg.Cycles) > 0 {
		logger.Log.Info("应用启动成功", map[string]interface{}{
			"cycles_count": len(config.Cfg.Cycles),
			"first_cycle":  config.Cfg.Cycles[0].Cycle,
		})
	} else {
		logger.Log.Warn("未配置任何交易周期")
	}

	// 获取所有交易对
	binanceFapi.GetSymbols()

	// 启动通知 Worker：消费队列，按订阅关系向用户发送 Telegram 消息
	go notify.StartWorker(ctx)

	// 启动费率周期更新 (独立于K线计算周期)
	go binanceFapi.GetRateCycle(ctx)

	// 启动MACD计算周期
	for _, c := range config.Cfg.Cycles {
		go calculate.MacdTicker(ctx, c.Cycle)
	}
	<-ctx.Done()
}

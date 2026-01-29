package main

import (
	"IndicatorTask/binanceFapi"
	"IndicatorTask/calculate"
	"IndicatorTask/config"
	"IndicatorTask/utils/logger"
	"context"
	"fmt"
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
	if err := database.AutoMigrate(&models.SymbolRecord{}); err != nil {
		panic("failed to migrate database: " + err.Error())
	}
	CleanNaNData()

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

	// 启动费率周期更新 (独立于K线计算周期)
	go binanceFapi.GetRateCycle(ctx)

	// 启动MACD计算周期
	for _, c := range config.Cfg.Cycles {
		go calculate.MacdTicker(ctx, c.Cycle)
	}
	<-ctx.Done()
}

// cleanNaNData 清理 symbol_records 中的非法数据 (NaN/Infinity) 与不准确 RSI，防止接口报错
func cleanNaNData() {
	cols := []string{"price", "volume", "taker_buy_volume", "taker_buy_ratio", "rsi", "rate", "change"}
	for _, col := range cols {
		sql := fmt.Sprintf("UPDATE symbol_records SET %s = 0 WHERE %s != %s OR %s::text = 'Infinity' OR %s::text = '-Infinity'", col, col, col, col, col)
		if err := database.DB.Exec(sql).Error; err != nil {
			fmt.Printf("Warning: Failed to clean NaN in %s: %v\n", col, err)
		}
	}
	database.DB.Exec("UPDATE symbol_records SET rsi = 50 WHERE rsi >= 99.99")
	fmt.Println("Database integrity check completed: NaN and inaccurate RSI records cleaned.")
}

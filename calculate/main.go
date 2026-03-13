package calculate

import (
	"IndicatorTask/binanceFapi"
	"IndicatorTask/config"
	"IndicatorTask/utils/logger"
	"IndicatorTask/utils/notify"

	"context"
	"time"

	"github.com/cryptoSelect/public/database"
	"github.com/cryptoSelect/public/models"
)

// 进行macd
func Start(ctx context.Context, cycle string) {
	// 增加延时，防止多周期协程同时操作 SymbolList 导致数据竞争
	delay := GetCycleDelay(cycle)
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Minute)
	}
	if len(binanceFapi.SymbolList) == 0 {
		binanceFapi.GetSymbols()
	}

	for _, symbolInfo := range binanceFapi.SymbolList {
		// 重置信号状态，确保每个周期和每一轮都是独立计算
		symbolInfo.CrossType = 0
		symbolInfo.Shape = 0
		symbolInfo.VpSignal = ""

		symbol := symbolInfo.Symbol
		Msg := ""

		// 获取K线数据
		klines, err := binanceFapi.GetContractKlines(symbol, cycle)
		if err != nil {
			logger.Log.Error("错误:", map[string]interface{}{"err": err})
			continue
		}
		if len(klines) < config.Cfg.Benchmark.Klines {
			logger.Log.Warn("数据不足", map[string]interface{}{"symbol": symbol, "count": len(klines), "required": config.Cfg.Benchmark.Klines})
			continue
		}
		lastKline := klines[config.Cfg.Benchmark.Klines-1:]
		if (time.Now().Unix() - lastKline[0].OpenTime) > 60*10 {
			lastKlineTime := time.Unix(lastKline[0].OpenTime/1000, 0).Format("2006-01-02 15:04:05")
			logger.Log.Warn("最后一根k线距当前时间超过10分钟，无效数据", map[string]interface{}{"time": lastKlineTime})
			continue
		}

		// 计算涨跌幅
		latestKline := klines[len(klines)-1]
		symbolInfo.Change = (latestKline.Close - latestKline.Open) / latestKline.Open * 100

		// 处理收线价格
		closes := binanceFapi.ClosePrice(klines)

		// 计算MACD (快线12，慢线26，信号线9)
		macd, signalLine, _ := calculateMACD(closes)

		// 计算交叉
		crossType, klineIndex := detectCrosses(klines, macd, signalLine)

		// 计算RSI
		rsiValue := GetRsi(closes)

		// MACD 交叉判断
		if klineIndex != 0 {
			symbolInfo.CrossType = crossType
		}

		// 计算缠论分型
		shape := detectFractal(klines)

		// symbolInfo 基础信息
		symbolInfo.Rsi = rsiValue
		symbolInfo.Rate = binanceFapi.GetRate(symbolInfo.Symbol)
		symbolInfo.Price = latestKline.Close
		takerBuyRatio := (latestKline.TakerBuyVolume / latestKline.Volume) * 100
		symbolInfo.Volume = latestKline.Volume
		symbolInfo.TakerBuyVolume = latestKline.TakerBuyVolume
		symbolInfo.TakerBuyRatio = takerBuyRatio

		// 量价分析
		symbolInfo.VpSignal = detectVolumePrice(klines, takerBuyRatio)

		// 将分析结果入库
		saveSymbolRecord(symbolInfo, cycle, klines, klineIndex)

		// 判定是否属于“异常”情况（满足任意一个则发通知）
		shouldNotify := false

		// 1. MACD 交叉
		if symbolInfo.CrossType != 0 {
			shouldNotify = true
		}

		// 2. 缠论分型
		if shape != 0 {
			symbolInfo.Shape = shape
			logger.Log.Info("缠论分型", map[string]interface{}{"symbol": symbolInfo.Symbol, "cycle": cycle, "shape": shape, "rsi": rsiValue})
			shouldNotify = true
		}

		// 3. RSI 超买超卖
		if rsiValue >= float64(config.Cfg.Benchmark.Rsi.Top) || rsiValue <= float64(config.Cfg.Benchmark.Rsi.Low) {
			shouldNotify = true
		}

		// 4. 量价异常 (背离、警惕、强势、恐慌、洗盘等)
		vp := symbolInfo.VpSignal
		if vp != "" && (contains(vp, "背离") || contains(vp, "强势") || contains(vp, "恐慌") || contains(vp, "洗盘") || contains(vp, "🔥")) {
			shouldNotify = true
		}

		if shouldNotify {
			Msg = alertMsgFmt(symbolInfo, cycle)
		}

		// 需要通知时入队，由 Worker 按订阅关系发送给对应用户
		if Msg != "" {
			notify.Push(symbol, cycle, Msg)
		}
	}

}

// 将分析结果入库及更新
func saveSymbolRecord(symbolInfo *binanceFapi.SymbolInfo, cycle string, klines []binanceFapi.KLine, klineIndex int) {
	updates := map[string]interface{}{
		"price":             symbolInfo.Price,
		"volume":            symbolInfo.Volume,
		"taker_buy_volume":  symbolInfo.TakerBuyVolume,
		"taker_buy_ratio":   symbolInfo.TakerBuyRatio,
		"rsi":               symbolInfo.Rsi,
		"rate":              symbolInfo.Rate,
		"rate_cycle":        symbolInfo.RateCycle,
		"cross_type":        symbolInfo.CrossType,
		"shape":             symbolInfo.Shape,
		"vp_signal":         symbolInfo.VpSignal,
		"change":            symbolInfo.Change,
		"next_funding_time": symbolInfo.NextFundingTime,
	}
	if klineIndex != 0 {
		updates["cross_time"] = time.Unix(klines[klineIndex].CloseTime/1000, 0)
	}

	result := database.DB.Model(&models.SymbolRecord{}).
		Where("symbol = ? AND cycle = ?", symbolInfo.Symbol, cycle).
		Updates(updates)
	if result.Error != nil || result.RowsAffected != 0 {
		return
	}

	rec := models.SymbolRecord{
		Symbol:          symbolInfo.Symbol,
		Cycle:           cycle,
		Price:           symbolInfo.Price,
		Volume:          symbolInfo.Volume,
		TakerBuyVolume:  symbolInfo.TakerBuyVolume,
		TakerBuyRatio:   symbolInfo.TakerBuyRatio,
		Rsi:             symbolInfo.Rsi,
		Rate:            symbolInfo.Rate,
		RateCycle:       symbolInfo.RateCycle,
		CrossType:       symbolInfo.CrossType,
		Shape:           symbolInfo.Shape,
		VpSignal:        symbolInfo.VpSignal,
		Change:          symbolInfo.Change,
		NextFundingTime: symbolInfo.NextFundingTime,
	}
	if klineIndex != 0 {
		rec.CrossTime = time.Unix(klines[klineIndex].CloseTime/1000, 0)
	}
	_ = database.DB.Create(&rec).Error
}

// ticker
func MacdTicker(ctx context.Context, cycle string) {
	duration := CycleDurationFmt(cycle)

	// 如果是开发模式，立即执行第一次
	if config.Cfg.Mode == "dev" {
		logger.Log.Info("开发模式: 立即开始首次执行", map[string]interface{}{"cycle": cycle})
		go Start(ctx, cycle)
	} else {
		// 计算距离下一次整点的时间
		now := time.Now()
		nextTick := now.Truncate(duration).Add(duration)
		waitTime := nextTick.Sub(now)

		logger.Log.Info("任务将在后开始", map[string]interface{}{"cycle": cycle, "time": nextTick.Format("15:04:05"), "wait": waitTime})

		// 等待第一次执行
		timer := time.NewTimer(waitTime)
		select {
		case <-ctx.Done():
			timer.Stop()
			logger.Log.Info("周期任务收到退出信号", map[string]interface{}{"cycle": cycle})
			return
		case <-timer.C:
			logger.Log.Info("周期性任务首次执行中...", map[string]interface{}{"cycle": cycle})
			go Start(ctx, cycle)
		}
	}

	// 启动周期性 Ticker
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("周期任务收到退出信号", map[string]interface{}{"cycle": cycle})
			return

		case <-ticker.C:
			logger.Log.Info("周期性任务执行中...", map[string]interface{}{"cycle": cycle})
			go Start(ctx, cycle)
		}
	}

}

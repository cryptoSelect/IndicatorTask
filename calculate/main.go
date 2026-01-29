package calculate

import (
	"IndicatorTask/binanceFapi"
	"IndicatorTask/config"
	"IndicatorTask/utils/logger"
	"IndicatorTask/utils/notify"

	"context"
	"time"
)

// 进行macd
func Start(ctx context.Context, cycle string) {
	// 增加延时，防止多周期协程同时操作 SymbolList 导致数据竞争
	delay := GetCycleDelay(cycle)
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Minute)
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

		// 判定是否属于“异常”情况（满足任意一个则发通知）
		shouldNotify := false

		// 1. MACD 交叉
		if symbolInfo.CrossType != 0 {
			shouldNotify = true
		}

		// 2. 缠论分型
		if shape != 0 {
			symbolInfo.Shape = shape
			// logger.Log.Info("缠论分型", map[string]interface{}{"symbol": symbolInfo.Symbol, "cycle": cycle, "shape": shape})
			shouldNotify = true
		}

		// 3. RSI 超买超卖
		if rsiValue >= float64(config.Cfg.Benchmark.Rsi.Top) || rsiValue <= float64(config.Cfg.Benchmark.Rsi.Low) {
			shouldNotify = true
		}

		// 4. 量价异常 (背离、警惕、强势、恐慌、洗盘等)
		vp := symbolInfo.VpSignal
		if vp != "" && (contains(vp, "背离") || contains(vp, "警惕") || contains(vp, "强势") || contains(vp, "恐慌") || contains(vp, "洗盘") || contains(vp, "🔥")) {
			shouldNotify = true
		}

		if shouldNotify {
			Msg = alertMsgFmt(symbolInfo, cycle)
		}

		// 出现分型立即通知
		if symbolInfo.Shape != 0 {
			notify.SendTelegramMessage(cycle, Msg)
		}

		if Msg == "" {
			continue
		}
	}

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

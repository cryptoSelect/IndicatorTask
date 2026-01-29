package calculate

import (
	"IndicatorTask/config"
	"math"
)

// 计算RSI
func GetRsi(prices []float64) float64 {
	period := config.Cfg.Benchmark.Rsi.Period

	// 数据不足，无法计算 RSI，返回 0 或者 50
	if len(prices) <= period {
		return 50
	}

	var gains, losses float64

	for i := 1; i <= period; i++ {
		diff := prices[i] - prices[i-1]
		if diff >= 0 {
			gains += diff
		} else {
			losses -= diff
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	for i := period + 1; i < len(prices); i++ {
		diff := prices[i] - prices[i-1]
		if diff >= 0 {
			avgGain = (avgGain*float64(period-1) + diff) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) - diff) / float64(period)
		}
	}

	// 如果没有上涨也没有下跌 (价格平盘)，RSI 应为 50
	if avgLoss == 0 && avgGain == 0 {
		return 50
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	// 最终检查 NaN
	if math.IsNaN(rsi) || math.IsInf(rsi, 0) {
		return 50
	}
	return rsi
}

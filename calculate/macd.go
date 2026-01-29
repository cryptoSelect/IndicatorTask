package calculate

import (
	"IndicatorTask/binanceFapi"
	"IndicatorTask/config"
)

// 计算EMA
func calculateEMA(prices []float64, period int) []float64 {
	ema := make([]float64, len(prices))
	k := 2.0 / float64(period+1)
	for i, price := range prices {
		if i == 0 {
			ema[i] = price
		} else {
			ema[i] = price*k + ema[i-1]*(1-k)
		}
	}
	return ema
}

// 计算MACD
func calculateMACD(closes []float64) ([]float64, []float64, []float64) {
	emaFast := calculateEMA(closes, config.Cfg.Benchmark.Macd.FastPeriod)
	emaSlow := calculateEMA(closes, config.Cfg.Benchmark.Macd.SlowPeriod)
	macd := make([]float64, len(closes))
	for i := range closes {
		macd[i] = emaFast[i] - emaSlow[i]
	}
	signalLine := calculateEMA(macd, config.Cfg.Benchmark.Macd.Window)
	histogram := make([]float64, len(closes))
	for i := range closes {
		histogram[i] = macd[i] - signalLine[i]
	}
	return macd, signalLine, histogram
}

// 检测金叉和死叉
func detectCrosses(klines []binanceFapi.KLine, macd, signalLine []float64) (int, int) {
	var index int
	var crossType int // 0: 无, 1: 金叉0轴上, 2: 金叉0轴下, 3: 死叉0轴上, 4: 死叉0轴下

	for i := 1; i < len(klines); i++ {
		prevMacd := macd[i-1]
		prevSignal := signalLine[i-1]
		currMacd := macd[i]
		currSignal := signalLine[i]

		if klines[i].Volume > 0 {
			// 金叉逻辑
			if prevMacd <= prevSignal && currMacd > currSignal {
				if currMacd > 0 {
					crossType = 1 // 金叉0轴上
				} else {
					crossType = 2 // 金叉0轴下
				}
				index = i
			}

			// 死叉逻辑
			if prevMacd >= prevSignal && currMacd < currSignal {
				if currMacd > 0 {
					crossType = 3 // 死叉0轴上
				} else {
					crossType = 4 // 死叉0轴下
				}
				index = i
			}
		}
	}

	if crossType != 0 {
		return crossType, index
	}
	return 0, 0
}

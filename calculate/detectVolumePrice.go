package calculate

import "IndicatorTask/binanceFapi"

// 检测量价关系
func detectVolumePrice(klines []binanceFapi.KLine, takerBuyRatio float64) string {
	n := len(klines)
	if n < 20 {
		return ""
	}

	currK := klines[n-1]
	prevK := klines[n-2]

	// 1. 计算均量 (20周期)
	var totalVol float64
	for i := n - 20; i < n; i++ {
		totalVol += klines[i].Volume
	}
	avgVol := totalVol / 20

	// 2. 判断价格趋势
	priceUp := currK.Close > prevK.Close
	priceDown := currK.Close < prevK.Close

	// 3. 判断成交量异动 (放量判断: 超过均量 1.5 倍)
	volSpike := currK.Volume > avgVol*1.5
	volShrink := currK.Volume < prevK.Volume*0.8 // 缩量: 比上一根少 20%

	var signal string

	if priceUp {
		if volSpike {
			signal = "放量上涨-强势"
			if takerBuyRatio > 55 {
				signal += " 🔥"
			}
		} else if volShrink {
			signal = "缩量上涨-背离"
		} else {
			signal = "放量齐升-健康"
		}
	} else if priceDown {
		if volSpike {
			signal = "放量下跌-恐慌"
		} else if volShrink {
			signal = "缩量下跌-洗盘"
		} else {
			signal = "放量下行-健康"
		}
	}

	// 4. 特殊情况: 放量滞涨
	if volSpike && !priceUp && !priceDown {
		signal = "放量滞涨-警惕"
	}

	return signal
}

func contains(s, substr string) bool {
	if substr == "" {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

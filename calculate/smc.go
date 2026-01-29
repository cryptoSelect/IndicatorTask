package calculate

import (
	"IndicatorTask/binanceFapi"
	"fmt"
	"math"
)

// SMCResult represents the technical levels identified by the SMC algorithm
type SMCResult struct {
	Support    float64
	Resistance float64
	Signal     string // BOS, CHoCH, or empty
	Fvg        string // FVG range
	Ob         string // OB range
}

// detectSMC 分析高低点、支撑压力位以及结构突破 (BOS/CHoCH)，并增加 FVG 和 OB 分析
func detectSMC(klines []binanceFapi.KLine) SMCResult {
	n := len(klines)
	if n < 30 {
		return SMCResult{}
	}

	// 1. 设置分析范围：最多最近 50 根 K 线
	lookback := 50
	if n < lookback {
		lookback = n
	}
	startIdx := n - lookback
	if startIdx < 0 {
		startIdx = 0
	}

	// 识别 Pivot High 和 Pivot Low (窗口大小为 5，表示 5+1+5 的结构)
	var pivotHighs []float64
	var pivotLows []float64

	window := 5
	// 我们从 startIdx 开始寻找在该 50 根范围内形成的 Pivot
	for i := startIdx; i < n-window; i++ {
		// 保证左侧有足够数据
		if i < window {
			continue
		}

		isHigh := true
		isLow := true
		for j := 1; j <= window; j++ {
			if klines[i].High < klines[i-j].High || klines[i].High < klines[i+j].High {
				isHigh = false
			}
			if klines[i].Low > klines[i-j].Low || klines[i].Low > klines[i+j].Low {
				isLow = false
			}
		}
		if isHigh {
			pivotHighs = append(pivotHighs, klines[i].High)
		}
		if isLow {
			pivotLows = append(pivotLows, klines[i].Low)
		}
	}

	res := SMCResult{}

	// 2. 提取最近 50 根内的最高压力和最低支撑 (即 SMC 的强区域)
	if len(pivotHighs) > 0 {
		maxHigh := pivotHighs[0]
		for _, v := range pivotHighs {
			if v > maxHigh {
				maxHigh = v
			}
		}
		res.Resistance = maxHigh
	}
	if len(pivotLows) > 0 {
		minLow := pivotLows[0]
		for _, v := range pivotLows {
			if v < minLow {
				minLow = v
			}
		}
		res.Support = minLow
	}

	// 3. FVG (Fair Value Gap) 检测
	// 扫描最近 20 根 K 线寻找最新的缺口
	for i := n - 1; i >= n-20 && i >= 2; i-- {
		// Bullish FVG: Low[i] > High[i-2]
		if klines[i].Low > klines[i-2].High && klines[i-1].Close > klines[i-1].Open {
			res.Fvg = "Bullish: [" + fmt.Sprintf("%.2f", klines[i-2].High) + " - " + fmt.Sprintf("%.2f", klines[i].Low) + "]"
			break
		}
		// Bearish FVG: High[i] < Low[i-2]
		if klines[i].High < klines[i-2].Low && klines[i-1].Close < klines[i-1].Open {
			res.Fvg = "Bearish: [" + fmt.Sprintf("%.2f", klines[i].High) + " - " + fmt.Sprintf("%.2f", klines[i-2].Low) + "]"
			break
		}
	}

	// 4. 结构突破判定 (BOS/CHoCH) 与 OB (Order Block)
	currClose := klines[n-1].Close
	prevClose := klines[n-2].Close

	if res.Resistance > 0 && currClose > res.Resistance && prevClose <= res.Resistance {
		res.Signal = "BOS-Bullish"
		// 寻找 OB: 在突破波段开始前的最后一根阴线
		for i := n - 2; i >= n-20 && i >= 0; i-- {
			if klines[i].Close < klines[i].Open {
				res.Ob = "Bullish OB: [" + fmt.Sprintf("%.2f", klines[i].Low) + " - " + fmt.Sprintf("%.2f", klines[i].High) + "]"
				break
			}
		}
	} else if res.Support > 0 && currClose < res.Support && prevClose >= res.Support {
		res.Signal = "BOS-Bearish"
		// 寻找 OB: 在跌破波段开始前的最后一根阳线
		for i := n - 2; i >= n-20 && i >= 0; i-- {
			if klines[i].Close > klines[i].Open {
				res.Ob = "Bearish OB: [" + fmt.Sprintf("%.2f", klines[i].Low) + " - " + fmt.Sprintf("%.2f", klines[i].High) + "]"
				break
			}
		}
	}

	return res
}

// 辅助函数
func getMin(a, b float64) float64 {
	return math.Min(a, b)
}

func getMax(a, b float64) float64 {
	return math.Max(a, b)
}

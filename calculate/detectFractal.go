package calculate

import "IndicatorTask/binanceFapi"

// K线对象简化版用于包含处理
type chanKLine struct {
	High float64
	Low  float64
}

// 检测分型 (缠论标准：包含处理 + 分型识别)
func detectFractal(rawKlines []binanceFapi.KLine) int {
	if len(rawKlines) < 5 {
		return 0
	}

	// 1. 包含处理
	processed := processInclusion(rawKlines)
	if len(processed) < 3 {
		return 0
	}

	// 我们对处理后的 K 线队列，取最后三根来判断分型
	n := len(processed)
	k1 := processed[n-3]
	k2 := processed[n-2]
	k3 := processed[n-1]

	// 顶分型：中间最高
	if k2.High > k1.High && k2.High > k3.High && k2.Low > k1.Low && k2.Low > k3.Low {
		return 1
	}

	// 底分型：中间最低
	if k2.High < k1.High && k2.High < k3.High && k2.Low < k1.Low && k2.Low < k3.Low {
		return 2
	}

	return 0
}

// 包含关系处理逻辑
func processInclusion(klines []binanceFapi.KLine) []chanKLine {
	if len(klines) == 0 {
		return nil
	}

	var res []chanKLine
	// 初始方向建议根据前两根判断，简化处理默认为向上
	isUp := true

	// 第一根入列
	res = append(res, chanKLine{High: klines[0].High, Low: klines[0].Low})

	for i := 1; i < len(klines); i++ {
		currH := klines[i].High
		currL := klines[i].Low
		last := res[len(res)-1]

		// 检查包含关系 (last 包含 curr 或 curr 包含 last)
		isIncluded := (last.High >= currH && last.Low <= currL) || (currH >= last.High && currL <= last.Low)

		if isIncluded {
			// 如果有包含关系，合并
			if isUp {
				// 向上：高高，低高
				newHigh := max(last.High, currH)
				newLow := max(last.Low, currL)
				res[len(res)-1] = chanKLine{High: newHigh, Low: newLow}
			} else {
				// 向下：低低，高低
				newHigh := min(last.High, currH)
				newLow := min(last.Low, currL)
				res[len(res)-1] = chanKLine{High: newHigh, Low: newLow}
			}
		} else {
			// 没有包含关系，判断新方向并加入
			if currH > last.High {
				isUp = true
			} else if currL < last.Low {
				isUp = false
			}
			res = append(res, chanKLine{High: currH, Low: currL})
		}
	}
	return res
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

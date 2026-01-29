package calculate

import (
	"IndicatorTask/binanceFapi"
	"IndicatorTask/config"
	"fmt"
	"strings"
	"time"
)

// 格式化为普通或“万”单位
func formatWithWan(val float64) string {
	if val >= 10000 {
		return fmt.Sprintf("%.2f万", val/10000)
	}
	return fmt.Sprintf("%.2f", val)
}

// 统一消息格式化
func alertMsgFmt(info *binanceFapi.SymbolInfo, cycle string) string {
	var builder strings.Builder

	// 1. 标题
	if info.Shape > 0 {
		var shapeStr string
		if info.Shape == 1 {
			shapeStr = "顶分型"
		} else if info.Shape == 2 {
			shapeStr = "底分型"
		}
		builder.WriteString(fmt.Sprintf("     ---- 【  %s %s %s 】 ---- \n", info.Symbol, cycle, shapeStr))
	} else {
		builder.WriteString(fmt.Sprintf("     ---- 【  %s %s  】 ---- \n", info.Symbol, cycle))
	}

	// 2. 基础信息
	builder.WriteString(fmt.Sprintf("价格: %.4f(%.2f%%)\n", info.Price, info.Change))

	// 3. 信号信息 (动态包含)
	// MACD 交叉
	// MACD 交叉
	if info.CrossType > 0 {
		var crossStr string
		switch info.CrossType {
		case 1:
			crossStr = "金叉0轴上"
		case 2:
			crossStr = "金叉0轴下"
		case 3:
			crossStr = "死叉0轴上"
		case 4:
			crossStr = "死叉0轴下"
		}
		builder.WriteString(fmt.Sprintf("MACD: %s\n", crossStr))
	}

	// 缠论分型
	if info.Shape > 0 {
		var shapeStr string
		if info.Shape == 1 {
			shapeStr = "顶分型"
		} else if info.Shape == 2 {
			shapeStr = "底分型"
		}
		builder.WriteString(fmt.Sprintf("形态: %s\n", shapeStr))
	}

	// RSI 状态
	rsiStatus := ""
	if info.Rsi >= float64(config.Cfg.Benchmark.Rsi.Top) {
		rsiStatus = " (超买)"
	} else if info.Rsi <= float64(config.Cfg.Benchmark.Rsi.Low) {
		rsiStatus = " (超卖)"
	}
	builder.WriteString(fmt.Sprintf("RSI: %.2f%s\n", info.Rsi, rsiStatus))

	// 成交信息
	builder.WriteString(fmt.Sprintf("成交: %s (%.2f%%)\n", formatWithWan(info.Volume), info.TakerBuyRatio))

	// 量价分析 (仅在不正常时显示)
	vp := info.VpSignal
	if vp != "" {
		builder.WriteString(fmt.Sprintf("量价: %s\n", vp))
	}

	// 4. 其他固定信息
	rateMsg := fmt.Sprintf("费率: %.4f%%", info.Rate)
	if info.NextFundingTime > 0 {
		hoursLeft := time.Until(time.Unix(info.NextFundingTime/1000, 0)).Hours()
		if hoursLeft > 0 {
			rateMsg += fmt.Sprintf(" (%.1fh结算)", hoursLeft)
		}
	}
	builder.WriteString(rateMsg + "\n")
	builder.WriteString(fmt.Sprintf("时间: %s", time.Now().Format("2006-01-02 15:04:05")))

	return builder.String()
}

// 判断使用小时周期还是分钟周期
func CycleDurationFmt(cycle string) time.Duration {
	var duration time.Duration
	switch cycle {
	case "5m":
		duration = 5 * time.Minute
	case "15m":
		duration = 15 * time.Minute
	case "30m":
		duration = 30 * time.Minute
	case "1h":
		duration = 1 * time.Hour
	case "4h":
		duration = 4 * time.Hour
	case "1d":
		duration = 24 * time.Hour
	case "1w":
		duration = 7 * 24 * time.Hour
	case "1M":
		duration = 30 * 24 * time.Hour
	default:
		duration = 30 * time.Minute
	}
	return duration
}

// 获取周期触发次数
func GetAlertCount(cycle string) int {
	for _, c := range config.Cfg.Cycles {
		if c.Cycle == cycle {
			return c.AlertCount
		}
	}
	// 默认5条
	return 5
}

// 获取周期延迟
func GetCycleDelay(cycle string) int {
	for _, c := range config.Cfg.Cycles {
		if c.Cycle == cycle {
			return c.DelayMinutes
		}
	}
	return 0
}

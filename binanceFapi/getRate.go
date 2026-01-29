package binanceFapi

import (
	"IndicatorTask/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cryptoSelect/public/database"
	"github.com/cryptoSelect/public/models"
	"github.com/ethereum/go-ethereum/log"
)

// 溢价指数回应-费率
type PremiumIndexResponse struct {
	Symbol          string  `json:"symbol"`
	LastFundingRate float64 `json:"lastFundingRate,string"`
	NextFundingTime int64   `json:"nextFundingTime"`
}

// 资金费率信息
type FundingInfo struct {
	Symbol               string `json:"symbol"`
	FundingIntervalHours int    `json:"fundingIntervalHours"`
}

// 获取费率
func GetRate(symbol string) float64 {
	url := fmt.Sprintf(config.Cfg.Api.Binance.FApi.Rate, symbol)
	resp, err := http.Get(url)
	if err != nil {
		log.Error("get rate failed: %s\nurl: %s\nerror: %s\n", url, err.Error())
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error("get rate parser request body error:", err.Error())
		return 0
	}

	// 这个接口返回的是单个对象
	var result PremiumIndexResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("rate Unmarshal failed: ", err.Error())
		log.Error("rate Unmarshal failed: ", err.Error())
		return 0
	}

	// 返回最新的 FundingRate
	return result.LastFundingRate
}

// 获取费率周期并更新数据库
func GetRateCycle(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		url := config.Cfg.Api.Binance.FApi.FundingInfo
		resp, err := http.Get(url)
		if err != nil {
			log.Error("get funding info failed: %s\nurl: %s\nerror: %s\n", url, err.Error())
			time.Sleep(1 * time.Minute)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close() // Close immediately after reading
		if err != nil {
			log.Error("read funding info body failed", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		var infos []FundingInfo
		if err := json.Unmarshal(body, &infos); err != nil {
			log.Error("unmarshal funding info failed", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		// 更新数据库
		for _, info := range infos {
			if info.FundingIntervalHours == 0 {
				continue
			}
			// 使用 map[string]interface{} 来只更新特定字段，避免覆盖其他字段
			if err := database.DB.Model(&models.SymbolRecord{}).
				Where("symbol = ?", info.Symbol).
				Update("rate_cycle", info.FundingIntervalHours).Error; err != nil {
				// 记录错误但不中断循环，可能是因为该 symbol 还没被插入到数据库中(例如不在监控列表)
				// log.Warn("update rate cycle failed", "symbol", info.Symbol, "err", err)
			}
		}

		// 批量同步更新内存中的数据
		mu.Lock()
		symbolMap := make(map[string]*SymbolInfo)
		for _, s := range SymbolList {
			symbolMap[s.Symbol] = s
		}
		for _, info := range infos {
			if s, ok := symbolMap[info.Symbol]; ok {
				s.RateCycle = info.FundingIntervalHours
			}
		}
		mu.Unlock()

		log.Info("Update RateCycle success")

		// 计算距离下一个整点的时间
		now := time.Now()
		nextHour := now.Truncate(time.Hour).Add(time.Hour)
		sleepDuration := time.Until(nextHour)

		time.Sleep(sleepDuration)

		// 每小时更新一次 symbol
		GetSymbols()
	}

}

package binanceFapi

import (
	"IndicatorTask/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type SymbolInfo struct {
	Symbol          string
	Price           float64
	Volume          float64
	TakerBuyVolume  float64
	TakerBuyRatio   float64
	Rsi             float64
	Rate            float64
	CrossType       int
	CrossTime       time.Time
	Shape           int
	VpSignal        string
	Change          float64
	NextFundingTime int64
	RateCycle       int
}

type SymbolPrice struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type SymbolChange struct {
	Symbol      string
	ClosePrice  float64
	CycleChange map[string]float64 // 周期 => 涨跌幅
}

type ChangeInfo struct {
	Change     float64
	ClosePrice float64
}

var (
	SymbolList []*SymbolInfo
	mu         sync.RWMutex
)

func GetSymbols() {
	resp, err := http.Get(config.Cfg.Api.Binance.FApi.Price)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var prices []SymbolPrice
	json.Unmarshal(body, &prices)

	var newList []*SymbolInfo
	for _, p := range prices {
		price, _ := strconv.ParseFloat(p.Price, 64)
		newList = append(newList, &SymbolInfo{Symbol: p.Symbol, Price: price})
	}

	mu.Lock()
	// 如果是第一次获取，直接赋值
	if len(SymbolList) == 0 {
		SymbolList = newList
	} else {
		// 如果不是第一次，保留原有的内存对象（为了保留里面的状态，虽然目前的架构中状态共享有问题）
		// 或者更好的做法是：如果是每小时更新，我们只需要添加新币种或更新价格
		symbolMap := make(map[string]*SymbolInfo)
		for _, s := range SymbolList {
			symbolMap[s.Symbol] = s
		}

		var updatedList []*SymbolInfo
		for _, ns := range newList {
			if existing, ok := symbolMap[ns.Symbol]; ok {
				existing.Price = ns.Price
				updatedList = append(updatedList, existing)
			} else {
				updatedList = append(updatedList, ns)
			}
		}
		SymbolList = updatedList
	}
	mu.Unlock()
}

// 安全获取 symbol 列表副本，防止遍历时发生竞态
func GetMonitoredSymbols() []*SymbolInfo {
	mu.RLock()
	defer mu.RUnlock()

	// 返回切片的副本
	cp := make([]*SymbolInfo, len(SymbolList))
	copy(cp, SymbolList)
	return cp
}

// 获取指定 symbol 和周期的涨跌幅
func GetChange(symbol, interval string) *ChangeInfo {
	url := fmt.Sprintf(config.Cfg.Api.Binance.FApi.Klines, symbol, interval, 2)
	resp, err := http.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var klines [][]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &klines); err != nil || len(klines) < 2 {
		return nil
	}

	open, _ := strconv.ParseFloat(klines[0][1].(string), 64)
	closeVal, _ := strconv.ParseFloat(klines[1][4].(string), 64)

	change := (closeVal - open) / open * 100
	return &ChangeInfo{Change: change, ClosePrice: closeVal}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

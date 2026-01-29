package binanceFapi

import (
	"IndicatorTask/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// K线数据结构
type KLine struct {
	OpenTime         int64   `json:"open_time"`
	Open             float64 `json:"open"`
	High             float64 `json:"high"`
	Low              float64 `json:"low"`
	Close            float64 `json:"close"`
	Volume           float64 `json:"volume"`
	CloseTime        int64   `json:"close_time"`
	QuoteVolume      float64 `json:"quote_volume"`
	NumTrades        int64   `json:"num_trades"`
	TakerBuyVolume   float64 `json:"taker_buy_volume"`
	TakerBuyQuoteVol float64 `json:"taker_buy_quote_volume"`
	Ignore           string  `json:"ignore"`
}

// 提取收盘价
func ClosePrice(klines []KLine) []float64 {
	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}
	return closes
}

// 获取K线数据
func GetContractKlines(symbol, cycle string) ([]KLine, error) {
	url := fmt.Sprintf(config.Cfg.Api.Binance.FApi.Klines, symbol, cycle, config.Cfg.Benchmark.Klines)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 400 {
			return nil, fmt.Errorf("Http.Code: %d, binance 没有该币合约: %s", resp.StatusCode, symbol)
		}
		return nil, fmt.Errorf("API请求失败，状态码：%d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawKlines [][]interface{}
	if err := json.Unmarshal(body, &rawKlines); err != nil {
		return nil, err
	}

	klines := make([]KLine, len(rawKlines))
	for i, k := range rawKlines {
		klines[i].OpenTime = int64(k[0].(float64))
		klines[i].Open, _ = strconv.ParseFloat(k[1].(string), 64)
		klines[i].High, _ = strconv.ParseFloat(k[2].(string), 64)
		klines[i].Low, _ = strconv.ParseFloat(k[3].(string), 64)
		klines[i].Close, _ = strconv.ParseFloat(k[4].(string), 64)
		klines[i].Volume, _ = strconv.ParseFloat(k[5].(string), 64)
		klines[i].CloseTime = int64(k[6].(float64))
		klines[i].QuoteVolume, _ = strconv.ParseFloat(k[7].(string), 64)
		klines[i].NumTrades = int64(k[8].(float64))
		klines[i].TakerBuyVolume, _ = strconv.ParseFloat(k[9].(string), 64)
		klines[i].TakerBuyQuoteVol, _ = strconv.ParseFloat(k[10].(string), 64)
		klines[i].Ignore = k[11].(string)
	}
	return klines, nil
}

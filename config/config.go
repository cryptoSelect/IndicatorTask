package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

var Cfg *ServerConfig

type ServerConfig struct {
	Mode      string           `json:"Mode"`
	Notify    Notify           `json:"Notify"`
	Api       Api              `json:"Api"`
	Cycles    []CycleThreshold `json:"Cycles"`
	Benchmark Benchmark        `json:"Benchmark"`
	Database  DBConfig         `json:"Database"`
}

type CycleThreshold struct {
	Cycle        string `json:"cycle"`
	AlertCount   int    `json:"AlertCount"`   // 周期内触发次数
	DelayMinutes int    `json:"DelayMinutes"` // 延时执行时间（分钟）
}

type DBConfig struct {
	Host     string `json:"Host"`
	Port     int    `json:"Port"`
	User     string `json:"User"`
	Password string `json:"Password"`
	DBName   string `json:"DBName"`
	SSLMode  string `json:"SSLMode"`
}

type Benchmark struct {
	Macd   Macd `json:"Macd"`
	Rsi    Rsi  `json:"Rsi"`
	Klines int  `json:"Klines"`
}

type Macd struct {
	FastPeriod int `json:"FastPeriod"`
	SlowPeriod int `json:"SlowPeriod"`
	Window     int `json:"Window"`
}

type Rsi struct {
	Top    int  `json:"Top"`
	Low    int  `json:"Low"`
	Period int  `json:"Period"`
	Enable bool `json:"Enable"`
}

type Notify struct {
	IsEnable         bool   `json:"IsEnable"`
	Token            string `json:"Token"`
	Group            string `json:"Group"`
	InformationTopic string `json:"InformationTopic"`
	Topic5Minue      string `json:"Topic5Minue"`
	Topic15Minue     string `json:"Topic15Minue"`
	Topic30Minue     string `json:"Topic30Minue"`
	Topic1Hour       string `json:"Topic1Hour"`
	Topic4Hour       string `json:"Topic4Hour"`
	Topic1Day        string `json:"Topic1Day"`
	Topic1Week       string `json:"Topic1Week"`
	Topic1Month      string `json:"Topic1Month"`
}

type Api struct {
	Binance     Binance     `json:"Binance"`
	TelegramBot TelegramBot `json:"TelegramBot"`
}

type TelegramBot struct {
	SentMsg string `json:"SentMsg"`
}

type Binance struct {
	Api  BinanceApi  `json:"Api"`
	FApi BinanceFApi `json:"FApi"`
}

type BinanceApi struct {
	Klines string `json:"Klines"`
	List   string `json:"List"`
}

type BinanceFApi struct {
	Price       string `json:"Price"`
	Klines      string `json:"Klines"`
	Rate        string `json:"Rate"`
	List        string `json:"List"`
	FundingInfo string `json:"FundingInfo"`
}

func LoadConfig(configNmae string) {
	wd, _ := os.Getwd()
	configPath := filepath.Join(wd, "config", configNmae)
	file, err := os.ReadFile(configPath)
	if err != nil {
		panic("config  file error:" + err.Error())
	}

	file = bytes.TrimPrefix(file, []byte("\xef\xbb\xbf"))
	var tmp ServerConfig
	if err := json.Unmarshal(file, &tmp); err != nil {
		panic("unmarshal json config err:" + err.Error())
	}
	Cfg = &tmp
}

func WatchConfig(configName string) {
	wd, _ := os.Getwd()
	configDir := filepath.Join(wd, "config")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic("failed to create watcher: " + err.Error())
	}
	defer watcher.Close()

	err = watcher.Add(configDir)
	if err != nil {
		panic("failed to watch config dir: " + err.Error())
	}

	fmt.Println(">>> Watching config dir:", configDir)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// 只处理目标 config 文件
			if filepath.Base(event.Name) == configName &&
				(event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Create == fsnotify.Create) {
				fmt.Println(">>> Config file modified:", event.Name)
				LoadConfig(configName)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("watch error:", err)
		}
	}
}

func Init() {
	LoadConfig("config.json")
	// go WatchConfig("config.json")
}

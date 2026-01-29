# IndicatorTask

Cryptocurrency selection analyzer — 指标与 K 线数据定时任务服务（Binance FAPI、MACD/RSI 等）。

**仓库地址：** <https://github.com/cryptoSelect/IndicatorTask.git>

```bash
git clone https://github.com/cryptoSelect/IndicatorTask.git
cd IndicatorTask
```

## 说明

- 依赖数据库（PostgreSQL）与 Binance FAPI，拉取 K 线、费率等数据，按配置周期计算 MACD、RSI 等指标并落库。
- 需配置 `config/config.json`（数据库、API、交易周期 `Cycles`、通知 Telegram 等），运行后拉取交易对、启动各周期 MACD 计算与费率更新任务。

## 本地运行

```bash
go mod download
# 配置 config/config.json 后
go run main/main.go
```

## Docker

```bash
# 构建（需先准备好 config/config.json）
docker build -t cryptoselect-indicatortask .

# 运行（挂载配置目录）
docker run --rm -v $(pwd)/config:/app/config cryptoselect-indicatortask
```

## License

MIT

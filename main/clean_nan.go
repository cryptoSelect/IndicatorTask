package main

import (
	"fmt"

	"github.com/cryptoSelect/public/database"
)

// cleanNaNData 清理 symbol_records 中的非法数据 (NaN/Infinity) 与不准确 RSI，防止接口报错
func CleanNaNData() {
	cols := []string{"price", "volume", "taker_buy_volume", "taker_buy_ratio", "rsi", "rate", "change"}
	for _, col := range cols {
		sql := fmt.Sprintf("UPDATE symbol_records SET %s = 0 WHERE %s != %s OR %s::text = 'Infinity' OR %s::text = '-Infinity'", col, col, col, col, col)
		if err := database.DB.Exec(sql).Error; err != nil {
			fmt.Printf("Warning: Failed to clean NaN in %s: %v\n", col, err)
		}
	}
	database.DB.Exec("UPDATE symbol_records SET rsi = 50 WHERE rsi >= 99.99")
	fmt.Println("Database integrity check completed: NaN and inaccurate RSI records cleaned.")
}

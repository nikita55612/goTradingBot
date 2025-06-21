package main_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nikita55612/goTradingBot/internal/broker/bybit"
)

func Test1(t *testing.T) {
	cli := bybit.NewClientFromEnv()
	ai, _ := cli.GetAccountInfo()
	fmt.Println(ai)
	for i := 3; i > 0; i-- {
		fmt.Println(i)
		time.Sleep(time.Second)
	}
	// n := 100.
	// s := []float64{1, 1.1, 1.2, 1.25}
	// sv := make([]float64, len(s))
	// // slices.Reverse(s)

	// for i := len(s) - 1; i >= 0; i-- {
	// 	sv[i] = n
	// 	n /= s[i]

	// 	fmt.Println(s[i])
	// }

	// fmt.Println(n)
	// fmt.Println(sv)
}

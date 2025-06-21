package main_test

import (
	"fmt"
	"testing"
	"time"
)

func Test1(t *testing.T) {
	var q string
	fmt.Scan(&q)
	if q == "q\n" {
		fmt.Println("is q")
	}
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

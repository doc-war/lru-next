package main

import (
	"fmt"
	"time"

	"github.com/doc-war/TTLCacheNext"
)

func main() {
	c, err := cache.New[string](1000, 5*time.Minute)
	if err != nil {
		panic(err)
	}

	val, err := c.GetOrLoad("hello", func(key string) (string, error) {
		return "world", nil
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(val) // world

	fmt.Println("Len:", c.Len()) // 1

	c.Delete("hello")
	fmt.Println("Len after delete:", c.Len()) // 0

	c.Clear()
}

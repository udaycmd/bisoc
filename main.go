package main

import "fmt"

func main() {
	a := []byte {'H', 'W', '2', '4'}
	for i := range 2 {
		fmt.Println("Hello, World", i, a[0:])
	}
}

package main

import (
	"fmt"

	"github.com/burke/mr232"
)

func main() {
	m, err := mr232.Start("/dev/ttys007")
	if err != nil {
		panic(err)
	}
	defer m.Close()

	go func() {
		for _ = range m.Lines {
		}
	}()

	a, b, c, d, e := m.GroupStatus(400)

	fmt.Println(a, b, c, d, e)
}

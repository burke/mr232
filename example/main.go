package main

import (
	"fmt"

	"github.com/burke/mr232"
)

func onAndBrightness(currLevel, lnzLevel int) (bool, int) {
	if currLevel == 0 {
		return false, lnzLevel
	}
	return true, currLevel
}

func main() {
	m, err := mr232.Start("/dev/ttys007")
	if err != nil {
		panic(err)
	}
	defer m.Close()

	go func() {
		for _ = range m.Messages {
		}
	}()

	gsm, err := m.GroupStatus(400)
	if err != nil {
		fmt.Println(err)
	}

	on, brightness := onAndBrightness(gsm.CurrentLevel, gsm.LastNonZeroLevel)
	fmt.Println(on, brightness)
}

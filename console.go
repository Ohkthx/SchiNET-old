package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func (b *bot) core() {
	var err error
	reader := bufio.NewReader(os.Stdin)

	channelsTemp()

	for {
		fmt.Printf("[%s] > ", time.Now().Format(time.Stamp))
		input, _ := reader.ReadString('\n')
		_, s := strToCommands(input)
		iodat := sliceToIOdat(b.Core, s)

		if len(iodat.io) > 0 {
			if iodat.io[0] == "exit" {
				break
			}
			err = iodat.ioHandler()
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	// Cleanup here from "exit"
	b.cleanup()
}

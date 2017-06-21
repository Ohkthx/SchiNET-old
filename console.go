package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func (cfg *Config) core() {
	var err error
	reader := bufio.NewReader(os.Stdin)

	//channelsTemp()

	for {
		fmt.Printf("[%s] > ", time.Now().Format(time.Stamp))
		input, _ := reader.ReadString('\n')
		_, s := strToCommands(input)
		iodat := sliceToIOdat(cfg.Core, s)

		if len(iodat.io) > 0 {
			if iodat.io[0] == "exit" {
				break
			} /*else if iodat.io[0] == "check" {
				fmt.Println("Got Check?")
				msg, err := cfg.MessageIntegrityCheck(iodat.io[1])
				if err != nil {
					fmt.Println(err)
					continue
				}
				fmt.Println(msg)
				continue
			}*/
			err = iodat.ioHandler()
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	// Cleanup here from "exit"
	cfg.cleanup()
}

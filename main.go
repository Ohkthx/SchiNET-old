package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/d0x1p2/godbot"
)

// Global struct to hold Connection data.
type Global struct {
	sync.Mutex
	*godbot.Connections
}

var (
	global   Global
	envToken = os.Getenv("DG_TOKEN")
)

func main() {

	if envToken == "" {
		return
	}

	b, err := godbot.New(envToken)
	if err != nil {
		fmt.Println(err)
		return
	}

	b.MessageHandler(msghandle)

	err = b.Start()
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Started...")
	global.Lock()
	global.Connections, err = b.GetConnections()
	global.Unlock()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Setup...")

	for {
	}

}

func msghandle(s *discordgo.Session, m *discordgo.MessageCreate) {
	if strings.Contains(m.Content, "list") {
		for i := 0; i < len(global.Channels); i++ {
			fmt.Println(global.Channels[i].Name)
		}
	}
}

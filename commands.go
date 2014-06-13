package main

import (
	"fmt"
)

//used for calling functions using string variable contents
type command func(chan string, string, string, string)

func initMap() map[string]command {
	return map[string]command{
		"source": command(source),
	}
}

func source(srvChan chan string, channel, nick, args string) {
	message := "PRIVMSG " + channel + " :https://github.com/heydabop/yaircb"
	fmt.Println(message)
	srvChan <- message
}

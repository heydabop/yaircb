package main

import(
	"fmt"
)

func source(srvChan chan string, channel, nick, args string){
	message := "PRIVMSG " + channel + " :https://github.com/heydabop/yaircb"
	fmt.Println(message)
	srvChan <- message
}

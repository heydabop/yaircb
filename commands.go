package main

import (
	"fmt"
)

//used for calling functions using string variable contents
type command func(chan string, string, string, string)

func initMap() map[string]command {
	return map[string]command{
		"source":   command(source),
		"botsnack": command(botsnack),
		"register": command(register),
	}
}

func source(srvChan chan string, channel, nick, args string) {
	message := "PRIVMSG " + channel + " :https://github.com/heydabop/yaircb"
	fmt.Println(message)
	srvChan <- message
}

func botsnack(srvChan chan string, channel, nick, args string) {
	message := "PRIVMSG " + channel + " :Kisses commend. Perplexities deprave."
	fmt.Println(message)
	srvChan <- message
}

func register(srvChan chan string, channel, nick, args string) {
	message := "PRIVMSG " + channel + " :http://anex.us:8080/register/"
	fmt.Println(message)
	srvChan <- message
}

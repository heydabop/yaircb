package main

import (
	"fmt"
	"os/exec"
	"strings"
)

//used for calling functions using string variable contents
type command func(chan string, string, string, string)

func initMap() map[string]command {
	return map[string]command{
		"source":   command(source),
		"botsnack": command(botsnack),
		"register": command(register),
		"uptime": command(uptime),
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
	message := "PRIVMSG " + channel + " :https://anex.us/register/"
	fmt.Println(message)
	srvChan <- message
}

func uptime(srvChan chan string, channel, nick, args string) {
	out, err := exec.Command("uptime").Output()
	message := "PRIVMSG " + channel + " :" + strings.TrimSpace(string(out))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(message)
	srvChan <- message
}

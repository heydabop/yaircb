package main

import (
	"fmt"
	"github.com/fzzy/radix/redis"
	"math/rand"
	"os/exec"
	"strings"
)

var cmdDb *redis.Client

//used for calling functions using string variable contents
type command func(chan string, string, string, string, []string)

func initMap() map[string]command {
	return map[string]command{
		"source":   command(source),
		"botsnack": command(botsnack),
		"register": command(register),
		"uptime":   command(uptime),
		"web":      command(web),
		"login":    command(login),
		"verify":   command(verify),
		"verified": command(verified),
		"help":     command(help),
		"commands": command(commands),
	}
}

func initCmdRedis() {
	cmdDb, _ = redis.Dial("tcp", "127.0.0.1:6379")
}

func source(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :https://github.com/heydabop/yaircb"
	fmt.Println(message)
	srvChan <- message
}

func botsnack(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :Kisses commend. Perplexities deprave."
	fmt.Println(message)
	srvChan <- message
}

func register(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :https://anex.us/register/"
	fmt.Println(message)
	srvChan <- message
}

func uptime(srvChan chan string, channel, nick, hostname string, args []string) {
	out, err := exec.Command("uptime").Output()
	message := "PRIVMSG " + channel + " :" + strings.TrimSpace(string(out))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(message)
	srvChan <- message
}

func web(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :https://anex.us/"
	fmt.Println(message)
	srvChan <- message
}

func login(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :https://anex.us/login/"
	fmt.Println(message)
	srvChan <- message
}

func verify(srvChan chan string, channel, nick, hostname string, args []string) {
	var message string
	if len(args) != 2 {
		message = "PRIVMSG " + channel + " :ERROR: Invalid number of arguments"
	} else {
		uname := args[0]
		pin := args[1]
		reply := cmdDb.Cmd("get", uname+"Pin")
		pinDb, _ := (reply.Bytes())
		if string(pinDb) == pin {
			message = "PRIVMSG " + channel + " :You are now verified as " + uname
			cmdDb.Cmd("set", uname+"Host", hostname)
			cmdDb.Cmd("set", uname+"Pin", fmt.Sprintf("%06d", rand.Intn(1000000)))
		} else {
			message = "PRIVMSG " + channel + " :PIN does not match that of " + uname
		}
	}
	fmt.Println(message)
	srvChan <- message
}

func verified(srvChan chan string, channel, nick, hostname string, args []string) {
	var message string
	if len(args) != 1 {
		message = "PRIVMSG " + channel + " :ERROR: Invalid number of arguments"
	} else {
		uname := args[0]
		reply := cmdDb.Cmd("get", uname+"Host")
		hostnameDb, _ := reply.Bytes()
		if hostname == string(hostnameDb) {
			message = "PRIVMSG " + channel + " :You are " + uname + " at " + hostname
		} else {
			message = "PRIVMSG " + channel + " :You are not " + uname
		}
	}
	fmt.Println(message)
	srvChan <- message
}

func help(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :8)"
	fmt.Println(message)
	srvChan <- message
}

func commands(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :"
	for command := range funcMap {
		message += command + " "
	}
	fmt.Println(message)
	srvChan <- message
}

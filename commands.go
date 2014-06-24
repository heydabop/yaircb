package main

import (
	"fmt"
	"github.com/fzzy/radix/redis"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strconv"
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
		"kick":     command(kick),
		"wc":       command(wc),
		"top":      command(top),
	}
}

func initCmdRedis() {
	var err error
	cmdDb, err = redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
		pinDb, err := (reply.Bytes())
		if err != nil {
			message = "PRIVMSG " + channel + " :" + fmt.Sprintf("%s", err)
		} else {
			if string(pinDb) == pin {
				message = "PRIVMSG " + channel + " :You are now verified as " + uname
				cmdDb.Cmd("set", uname+"Host", hostname)
				cmdDb.Cmd("set", uname+"Pin", fmt.Sprintf("%06d", rand.Intn(1000000)))
			} else {
				message = "PRIVMSG " + channel + " :PIN does not match that of " + uname
			}
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
		hostnameDb, err := reply.Bytes()
		if err != nil {
			message = "PRIVMSG " + channel + " :" + fmt.Sprintf("%s", err)
		} else {
			if hostname == string(hostnameDb) {
				message = "PRIVMSG " + channel + " :You are " + uname + " at " + hostname
			} else {
				message = "PRIVMSG " + channel + " :You are not " + uname
			}
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

func kick(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "KICK " + channel + " " + nick + " :You don't tell me what to do."
	fmt.Println(message)
	srvChan <- message

	message = "KICK " + channel
	if len(args) < 1 {
		message = "PRIVMSG " + channel + " :ERROR: Invalid number of arguments"
	} else {
		if args[0] == config.Nick {
			return
		}
		message += " " + args[0]
	}
	if len(args) >= 2 {
		message += " :" + strings.Join(args[1:], " ")
	}
	fmt.Println(message)
	srvChan <- message
}

func wc(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :"
	if len(args) != 1 {
		message += "ERROR: Invalid number of arguments"
	} else {
		logFile, err := os.Open(`/home/ross/irclogs/freenode/` + channel + `.log`)
		if err != nil {
			message += fmt.Sprintf("%s", err)
		} else {
			fileStat, err := logFile.Stat()
			if err != nil {
				message += fmt.Sprintf("%s", err)
			} else {
				log := make([]byte, fileStat.Size())
				_, err = logFile.Read(log)
				if err != nil {
					message += fmt.Sprintf("%s", err)
				} else {
					logLines := strings.Split(string(log), "\n")
					nickLine := regexp.MustCompile(`^\d\d:\d\d <[@\+\s]?` + args[0] + `>`)
					matches := 0
					for _, line := range logLines {
						if match := nickLine.FindStringSubmatch(line); match != nil {
							matches++
						}
					}
					message += args[0] + ": " + fmt.Sprintf("%d", matches) + " lines"
				}
			}
		}
	}
	fmt.Println(message)
	srvChan <- message
}

func top(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :"
	if len(args) != 1 {
		message += "ERROR: Invalid number of arguments"
	} else {
		nicks64, err := strconv.ParseInt(args[0], 10, 0)
		if err != nil {
			message += fmt.Sprintf("%s", err)
		} else {
			nicks := int(nicks64)
			logFile, err := os.Open(`/home/ross/irclogs/freenode/` + channel + `.log`)
			if err != nil {
				message += fmt.Sprintf("%s", err)
			} else {
				fileStat, err := logFile.Stat()
				if err != nil {
					message += fmt.Sprintf("%s", err)
				} else {
					log := make([]byte, fileStat.Size())
					_, err = logFile.Read(log)
					if err != nil {
						message += fmt.Sprintf("%s", err)
					} else {
						logLines := strings.Split(string(log), "\n")
						nickLine := regexp.MustCompile(`^\d\d:\d\d <[@\+\s]?(\S*?)>`)
						matches := make(map[string]uint)
						for _, line := range logLines {
							if match := nickLine.FindStringSubmatch(line); match != nil {
								matches[strings.ToLower(match[1])]++
							}
						}
						for i := 0; i < nicks; i++ {
							maxLines := uint(0)
							var maxNick string
							for nick, lines := range matches {
								if lines > maxLines {
									maxLines = lines
									maxNick = nick
								}
							}
							if maxLines < 1 {
								break
							}
							message += string(maxNick[0]) + string('\u200B') + maxNick[1:] + ": " + fmt.Sprintf("%d", maxLines) + " lines || "
							delete(matches, maxNick)
						}
					}
				}
			}
		}
	}
	fmt.Println(message)
	srvChan <- message
}

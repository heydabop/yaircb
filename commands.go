package main

import (
	"encoding/json"
	"fmt"
	"github.com/fzzy/radix/redis"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var cmdDb *redis.Client

//command is the format for all bot command functions. The chan string is used to send generated output to the server;
//the first string is the channel from which the command is called and reply is sent to;
//the second string is the nick that called the command;
//the third string is the hostname of the user that called the command;
//and the []string contains any arguments to the command.
//All commands direct any output to both chan string (the IRC server) and console.
type command func(chan string, string, string, string, []string)

//initMap is used to populate the global variable funcMap in irc.go,
//it allows calling functions based upon strings.
func initMap() map[string]command {
	return map[string]command{
		"source":    command(source),
		"botsnack":  command(botsnack),
		"register":  command(register),
		"uptime":    command(uptime),
		"web":       command(web),
		"login":     command(login),
		"verify":    command(verify),
		"verified":  command(verified),
		"help":      command(help),
		"commands":  command(commands),
		"kick":      command(kick),
		"wc":        command(wc),
		"top":       command(top),
		"footprint": command(footprint),
		"commit":    command(commit),
	}
}

//initCmdRedis initializes the global variable cmdDb via the local database
func initCmdRedis() {
	var err error
	cmdDb, err = redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

//source outputs a link to the repository on github
func source(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :https://github.com/heydabop/yaircb"
	fmt.Println(message)
	srvChan <- message
}

//botsnack outputs a pointless message
func botsnack(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :Kisses commend. Perplexities deprave."
	fmt.Println(message)
	srvChan <- message
}

//register outputs a link to register with the webserver
func register(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :https://anex.us/register/"
	fmt.Println(message)
	srvChan <- message
}

//uptime outputs the command 'uptime'
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

//web outputs a link to the homepage of the webserver
func web(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :https://anex.us/"
	fmt.Println(message)
	srvChan <- message
}

//login outputs a link to the login page of the webserver
func login(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :https://anex.us/login/"
	fmt.Println(message)
	srvChan <- message
}

//verify takes two arguments, the first being a username, the second being a PIN associated to that username.
//verify <username> <pin>
//If the username and PIN match those displayed on a user page on the webserver, then the IRC nick@hostname and webserver
//username become associated to each other.
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

//verified takes one argument, the username against which the IRC user is testing association
//verified <username>
//If the IRC nick@hostname is associated to the webserver username, that state is indicated by the bot's response.
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

//help is currently unhelpful
func help(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :8)"
	fmt.Println(message)
	srvChan <- message
}

//commands outputs every publicly callable command
func commands(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :"
	for command := range funcMap {
		message += command + " "
	}
	fmt.Println(message)
	srvChan <- message
}

//kick takes at least one and up to two arguments
//kick <nick> <reason>
//If the bot has OP, nick is kicked with reason. The caller of the command is held responsible...
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

//wc takes one argument, the user who's messages are being counted
//wc <nick>
//wc outputs the number of messages nick has said in channel
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

//top takes one argument, the number of nick line counts to output
//top <n>
//top outputs the most active n users, by outputting their nicks and the number of messages in channel
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

//yesNo is not called like other commands, and is instead instantiated when a message starts with the bot name and ends
//with a question mark.
//yesNo randomly outputs "Yes." or "No."
func yesNo(srvChan chan string, channel, nick, hostname string) {
	message := "PRIVMSG " + channel + " :"
	x := rand.Intn(2)
	if x == 1 {
		message += "Yes."
	} else {
		message += "No."
	}
	fmt.Println(message)
	srvChan <- message
}

//footprint outputs the resident memory usage of the process
func footprint(srvChan chan string, channel, nick, hostname string, args []string) {
	message := "PRIVMSG " + channel + " :"
	pid := os.Getpid()
	out, err := exec.Command("grep", "VmRSS", "/proc/"+fmt.Sprintf("%d", pid)+"/status").Output()
	if err != nil {
		message += fmt.Sprintf("%s", err)
	} else {
		kbRegex := regexp.MustCompile(`VmRSS:\s*(.*)`)
		if match := kbRegex.FindStringSubmatch(string(out)); match != nil {
			message += strings.TrimSpace(match[1])
		}
	}
	srvChan <- message
	fmt.Println(message)
}

//commit randomly selects a github repository and commit and outputs the first line of the commit
//and a goo.gl URL of the commit
func commit(srvChan chan string, channel, nick, hostname string, args []string) {
	type repoJSON struct {
		Id          int
		Owner       map[string]interface{}
		Name        string
		Full_name   string
		Description string
		Private     bool
		Fork        bool
		Url         string
		Html_url    string
	}
	type commitJSON struct {
		Sha          string
		Commit       map[string]interface{}
		Url          string
		Html_url     string
		Comments_url string
		Author       map[string]interface{}
		Committer    map[string]interface{}
		Parents      map[string]interface{}
	}
	type urlJSON struct {
		Kind    string
		Id      string
		LongUrl string
	}
	message := "PRIVMSG " + channel + " :"
	since := rand.Intn(1000000)
	res, err := http.Get("https://api.github.com/repositories?since=" + fmt.Sprintf("%d", since))
	if err != nil {
		message += fmt.Sprintf("%s", err)
	} else {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			message += fmt.Sprintf("%s", err)
		} else {
			var repos []repoJSON
			json.Unmarshal(body, &repos)
			fullName := repos[rand.Intn(len(repos))].Full_name
			res, err = http.Get("https://api.github.com/repos/" + fullName + "/commits")
			if err != nil {
				message += fmt.Sprintf("%s", err)
			} else {
				body, err = ioutil.ReadAll(res.Body)
				if err != nil {
					message += fmt.Sprintf("%s", err)
				}
				var commits []commitJSON
				json.Unmarshal(body, &commits)
				if len(commits) < 1 {
					message += "ERROR: No commits in selected repository"
				} else {
					commitNum := rand.Intn(len(commits))
					commitMsg := commits[commitNum].Commit["message"].(string)

					urlReader := strings.NewReader(`{"longUrl": "` + commits[commitNum].Html_url + `"}`)
					c := http.Client{}
					res, err := c.Post("https://www.googleapis.com/urlshortener/v1/url", "application/json", urlReader)
					if err != nil {
						message += fmt.Sprintf("%s", err)
					} else {
						body, err := ioutil.ReadAll(res.Body)
						if err != nil {
							message += fmt.Sprintf("%s", err)
						} else {
							var googUrl urlJSON
							json.Unmarshal(body, &googUrl)
							message += strings.Split(commitMsg, "\n")[0] + " | " + googUrl.Id
						}
					}
				}
			}
		}
	}
	srvChan <- message
	fmt.Println(message)
}

package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/textproto"
	"os"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var (
	funcMap    map[string]command
	regexpCmds []*regexp.Regexp
	config     JSONconfig
)

type JSONconfig struct {
	Server       string
	Port         int
	Nick         string
	NickServPass string
	Hostname     string
	TLS          bool
	Admins       []string
}

//output err
func errOut(err error, quitChans chan chan bool) {
	log.Println("ERROR: ", err.Error())
	debug.PrintStack()
	log.Println("EXITING")
chanLoop:
	for {
		select {
		case quit := <-quitChans:
			quit <- true
			close(quit)
		default:
			break chanLoop
		}
	}
	log.Println("QUITS SENT")
}

//take input from srvChan and send to server
func writeToServer(w *bufio.Writer, srvChan chan string, wg *sync.WaitGroup, quit chan bool, quitChans chan chan bool) {
	defer wg.Done()
	defer fmt.Println("WTS") //debug

	_, err := w.WriteString("PING" + config.Nick + "\r\n") //test message. primarily to get to select loop
	if err == nil {
		err = w.Flush()
	}
	//send all lines in srvChan to server
	for err == nil {
		select {
		case <-quit: //exit if indicated
			return
		case str := <-srvChan:
			_, err = w.WriteString(str + "\r\n")
			if err == nil {
				err = w.Flush()
			}
		}
	}

	//print error and exit
	if err != nil {
		errOut(err, quitChans)
	}
}

//take input from connection and send to console channel
func readFromServer(r *bufio.Reader, srvChan chan string, wg *sync.WaitGroup, quit chan bool, quitChans chan chan bool) {
	defer wg.Done()
	defer fmt.Println("RFS")

	line, line_err := r.ReadString('\n')
	for ; line_err == nil; line, line_err = r.ReadString('\n') {
		select {
		case <-quit:
			return
		default:
			srvChan <- strings.TrimSpace(line)
		}
	}
	if line_err != nil {
		errOut(line_err, quitChans)
	}
}

func writeToConsole(readChan chan string, writeChan chan string, wg *sync.WaitGroup, quit chan bool, quitChans chan chan bool) {
	defer wg.Done()
	defer fmt.Println("WTC") //debug

	pingRegex := regexp.MustCompile("^PING (.*)")
	questionRegex := regexp.MustCompile(`^:(\S*?)!(\S*?)@(\S*?) PRIVMSG (\S*) :` + config.Nick + `.*\?`)
	ctcpRegex := regexp.MustCompile(`^:(\S*?)!(\S*?)@(\S*?) PRIVMSG (\S*) :` + "\x01" + `(.*?)` + "\x01" + `$`)

	//read every line from the server chan and print to console
	for {
		select {
		case <-quit: //exit if indicated
			return
		case line := <-readChan:
			log.Println(line)
			if match := pingRegex.FindStringSubmatch(line); match != nil {
				//respond to PING from server
				writeChan <- ("PONG " + match[1])
				log.Println("PONG", match[1])
			} else if match := questionRegex.FindStringSubmatch(line); match != nil {
				go yesNo(writeChan, match[4], match[1], match[3]) //reply Yes or No if bot was asked a question
			} else if match := ctcpRegex.FindStringSubmatch(line); match != nil {
				go ctcp(writeChan, match[4], match[1], match[3], strings.Fields(match[5])) //reply with CTCP if CTCP request was received
			} else {
				var match []string
				for _, regexp := range regexpCmds {
					if match = regexp.FindStringSubmatch(line); match != nil {
						cmdArgs := strings.Fields(match[5]) //first word is command, the rest (if any) are args for the command
						if cmd, valid := funcMap[strings.ToLower(cmdArgs[0])]; valid {
							if match[4] == config.Nick {
								match[4] = match[1]
							}
							go cmd(writeChan, match[4], match[1], match[3], cmdArgs[1:])
						}
						break
					}
				}
			}
		}
	}
}

//read input from console and send to srvChan
func readFromConsole(srvChan chan string, wg *sync.WaitGroup, error chan bool, quitChans chan chan bool) {
	defer wg.Done()
	defer fmt.Println("RFC") //debug

	in := bufio.NewScanner(os.Stdin)
	//read all text from console, send it to srvChan to be sent to server
	for in.Scan() {
		str := in.Text()
		srvChan <- str
		if strings.TrimSpace(str) == "QUIT" { //exit and indicate upon reading QUIT
			error <- true
			return
		}
	}

	//print error and exit
	if err := in.Err(); err != nil {
		error <- true
		errOut(err, quitChans)
	}
}

func main() {
	runtime.GOMAXPROCS(4)
	rand.Seed(time.Now().Unix())
	//read in bot config, or initialize default config
	configFile, err := ioutil.ReadFile("config.json")
	if err == nil {
		json.Unmarshal(configFile, &config)
	} else {
		config = JSONconfig{"chat.freenode.net", 6697, "yaircb", "", "*", false, make([]string, 0)}
	}

	//set up command detection regular expressions
	regexpCmds = make([]*regexp.Regexp, 3)
	regexpCmds[0] = regexp.MustCompile(`^:(\S*?)!(\S*?)@(\S*?) PRIVMSG (\S*) :` + config.Nick + `\W?\s*(.*)`)
	regexpCmds[1] = regexp.MustCompile(`^:(\S*)?!(\S*)?@(\S*)? PRIVMSG (\S*) :\s*\+(.*)`)
	regexpCmds[2] = regexp.MustCompile(`^:(\S*)?!(\S*)?@(\S*)? PRIVMSG (` + config.Nick + `) :\s*(.*)`)

	//initialize global string->function command map
	funcMap = initMap()
	initCmdRedis()

	//initialize web server
	initWebRedis()
	http.Handle("/resources/", http.StripPrefix("/resources/", http.FileServer(http.Dir("resources"))))
	http.HandleFunc("/register/", registerHandler)
	http.HandleFunc("/login/", loginHandler)
	http.HandleFunc("/loginCheck/", loginCheckHandler)
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/save/", saveHandler)
	http.HandleFunc("/user/", userHandler)
	go http.ListenAndServeTLS(":8080", "ssl.crt", "ssl.pem", nil)

	var conns uint16
	writeChan := make(chan string) //used to send strings from readFromConsole to writeToServer
	readChan := make(chan string)  //send strings from readFromServer to writeToConsole
	//wgSrv for goroutines to/from sever, wg for readFromConsole
	var wgSrv, wg sync.WaitGroup
	quitChans := make(chan chan bool, 2)
	error := make(chan bool, 1) //used to indicate readFromConsole exited
	//initiate connection
	wg.Add(2)
	go readFromConsole(writeChan, &wg, error, quitChans) //doesnt get restarted on connection EOF
	wtsQChan := make(chan bool, 1)
	go writeToConsole(readChan, writeChan, &wg, wtsQChan, quitChans) //doesnt get restarted on connection EOF
connectionLoop:
	for ; ; conns++ {
		select {
		case <-error: //if readFromConsole got a "QUIT", exit program
			wtsQChan <- true
			break connectionLoop
		default: //otherwise restart connections
			if conns == 0 {
				log.Println("STARTING...")
			} else {
				log.Println("RESTARTING...")
				log.Println("WAITING 1 MINUTE...")
				time.Sleep(time.Minute)
			}
			var socketRead *bufio.Reader
			var socketWrite *bufio.Writer
			err := errors.New("")
			if config.TLS {
				log.Printf("Connecting to %s:%d with TLS...\n", config.Server, config.Port)
				var sslSocket *tls.Conn
				sslSocket, err = tls.Dial("tcp", fmt.Sprintf("%s:%d", config.Server, config.Port), nil)
				if err == nil {
					sslSocket.SetReadDeadline(time.Time{})
					socketWrite = bufio.NewWriter(sslSocket)
					socketRead = bufio.NewReader(sslSocket)
				} else {
					log.Println("Disabling TLS...")
				}
			}
			if err != nil || !config.TLS { //!config.TLS shouldn't actually be evaluted, as err != nil would be true from err init
				log.Printf("Connecting to %s:%d...\n", config.Server, config.Port)
				socket, err := textproto.Dial("tcp", fmt.Sprintf("%s:%d", config.Server, config.Port))
				if err != nil {
					errOut(err, quitChans)
					return
				}
				socketWrite = socket.Writer.W
				socketRead = socket.Reader.R
			}
			//make writer/reader to/from server
			//send initial IRC messages, NICK and USER
			_, err = socketWrite.WriteString("NICK " + config.Nick + "\r\n")
			if err == nil {
				err = socketWrite.Flush()
			}
			log.Print("NICK " + config.Nick + "\r\n")
			if err != nil {
				errOut(err, quitChans)
			}
			_, err = socketWrite.WriteString("USER " + config.Nick + " " + config.Hostname + " * :yaircb\r\n")
			if err == nil {
				err = socketWrite.Flush()
			}
			log.Print("USER " + config.Nick + " " + config.Hostname + " * :yaircb\r\n")
			if err != nil {
				errOut(err, quitChans)
			}
			wgSrv.Add(1)
			//launch routine to write server output to console
			rfsQChan := make(chan bool, 1)
			quitChans <- rfsQChan
			go readFromServer(socketRead, readChan, &wgSrv, rfsQChan, quitChans)
			//join first channel
			/*err = socket.Writer.PrintfLine("JOIN #ttestt")
			if err != nil {
				errOut(err, quit)
			}*/
			if config.NickServPass != "" {
				_, err = socketWrite.WriteString("PRIVMSG NickServ :IDENTIFY " + config.NickServPass + "\r\n")
				if err == nil {
					err = socketWrite.Flush()
				}
				if err != nil {
					errOut(err, quitChans)
				}
			}
			wgSrv.Add(1)
			//launch routine to send to server and get input from console
			wtsQChan := make(chan bool, 1)
			quitChans <- wtsQChan
			go writeToServer(socketWrite, writeChan, &wgSrv, wtsQChan, quitChans)
			wgSrv.Wait()
		}
	}
	wg.Wait()
}

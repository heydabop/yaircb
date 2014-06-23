package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/fzzy/radix/redis"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	funcMap    map[string]command
	db         redis.Client
	regexpCmds []*regexp.Regexp
	config     JSONconfig
)

type JSONconfig struct {
	Nick string
	Pass string
	Hostname string
}

//output err
func errOut(err error, quit chan bool) {
	fmt.Println("ERROR: ", err.Error())
	var trace []byte
	runtime.Stack(trace, false)
	fmt.Print(trace)
	if err.Error() == "EOF" {
		fmt.Println("EXITING")
		for i := 0; i < 2; i++ {
			quit <- true
		}
		fmt.Println("QUITS SENT")
	}
}

//take input from srvChan and send to server
func writeToServer(socket *tls.Conn, srvChan chan string, wg *sync.WaitGroup, quit chan bool) {
	defer wg.Done()
	defer fmt.Println("WTS") //debug

	w := bufio.NewWriter(socket)
	_, err := w.WriteString(<-srvChan + "\r\n")
	//send all lines in srvChan to server
	for err == nil {
		select {
		case <-quit: //exit if indicated
			return
		case str := <-srvChan:
			_, err = w.WriteString(str + "\r\n")
		}
	}

	//print error and exit
	if err != nil {
		errOut(err, quit)
		socket.Close()
	}
}

//take input from connection and send to console channel
func readFromServer(socket *tls.Conn, srvChan chan string, wg *sync.WaitGroup, quit chan bool) {
	defer wg.Done()
	defer fmt.Println("RFS")

	r := bufio.NewReader(socket)
	line, line_err := r.ReadString('\n')
	for ; line_err == nil; line, line_err = r.ReadString('\n'){
		select {
		case <-quit:
			return
		default:
			srvChan <- line
		}
	}
	if line_err != nil {
		errOut(line_err, quit)
		socket.Close()
	}
}

func writeToConsole(readChan chan string, writeChan chan string, wg *sync.WaitGroup, quit chan bool) {
	defer wg.Done()
	defer fmt.Println("WTC") //debug

	pingRegex := regexp.MustCompile("^PING (.*)")

	//read every line from the server chan and print to console
	for {
		select {
		case <-quit: //exit if indicated
			return
		case line := <-readChan:
			fmt.Println(line)
			if match := pingRegex.FindStringSubmatch(line); match != nil {
				//respond to PING from server
				writeChan <- ("PONG " + match[1])
				fmt.Println("PONG", match[1]) //put to console
			} else {
				var match []string
				for _, regexp := range regexpCmds {
					if match = regexp.FindStringSubmatch(line); match != nil {
						cmdArgs := strings.Fields(match[5])
						if cmd, valid := funcMap[cmdArgs[0]]; valid {
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
func readFromConsole(srvChan chan string, wg *sync.WaitGroup, quit chan bool, error chan bool) {
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
		errOut(err, quit)
	}
}

func main() {
	rand.Seed(time.Now().Unix())
	configFile, err := ioutil.ReadFile("config.json")
	if err == nil {
		json.Unmarshal(configFile, &config)
	} else {
		config = JSONconfig{"yaircb", "", "*"}
	}

	regexpCmds = make([]*regexp.Regexp, 3)
	regexpCmds[0] = regexp.MustCompile(`^:(\S*?)!(\S*?)@(\S*?) PRIVMSG (\S*) :` + config.Nick + `:\s*(.*)`)
	regexpCmds[1] = regexp.MustCompile(`^:(\S*)?!(\S*)?@(\S*)? PRIVMSG (\S*) :\s*\+(.*)`)
	regexpCmds[2] = regexp.MustCompile(`^:(\S*)?!(\S*)?@(\S*)? PRIVMSG (` + config.Nick + `) :\s*(.*)`)

	funcMap = initMap()
	initCmdRedis()

	db, err := redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println(err)
		return
	}
	reply, err := db.Cmd("get", "heydabop").Bytes()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(reply))

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
	quit := make(chan bool, 2)  //used to indicate server to/from goroutines need to exit
	error := make(chan bool, 1) //used to indicate readFromConsole exited
	//initiate connection
	wg.Add(2)
	go readFromConsole(writeChan, &wg, quit, error)   //doesnt get restarted on connection EOF
	go writeToConsole(readChan, writeChan, &wg, quit) //doesnt get restarted on connection EOF
connectionLoop:
	for ; ; conns++ {
		select {
		case <-error: //if readFromConsole got a "QUIT", exit program
			break connectionLoop
		default: //otherwise restart connections
			if conns == 0 {
				fmt.Println("STARTING...")
			} else {
				fmt.Println("RESTARTING...")
			}
			socket, err := tls.Dial("tcp", "chat.freenode.net:6697", nil)
			if err != nil {
				errOut(err, quit)
				return
			}
			//make writer/reader to/from server
			//send initial IRC messages, NICK and USER
			w := bufio.NewWriter(socket)
			_, err = w.WriteString("USER " + config.Nick + " " + config.Hostname + " * :yaircb\r\n")
			fmt.Print("USER " + config.Nick + " " + config.Hostname + " * :yaircb\r\n")
			if err != nil {
				errOut(err, quit)
			}
			_, err = w.WriteString("NICK " + config.Nick + "\r\n")
			fmt.Print("NICK " + config.Nick + "\r\n")
			if err != nil {
				errOut(err, quit)
			}
			wgSrv.Add(1)
			//launch routine to write server output to console
			go readFromServer(socket, readChan, &wgSrv, quit)
			//join first channel
			/*err = socket.Writer.PrintfLine("JOIN #ttestt")
			if err != nil {
				errOut(err, quit)
			}*/
			_, err = w.WriteString("PRIVMSG NickServ :IDENTIFY " + config.Pass + "\r\n")
			wgSrv.Add(1)
			//launch routine to send to server and get input from console
			go writeToServer(socket, writeChan, &wgSrv, quit)
			wgSrv.Wait()
		}
	}
	wg.Wait()
}

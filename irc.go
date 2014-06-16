package main

import (
	"bufio"
	"fmt"
	"github.com/fzzy/radix/redis"
	"net/http"
	"net/textproto"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

var (
	funcMap map[string]command
	db      redis.Client
)

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
func writeToServer(socket *textproto.Conn, srvChan chan string, wg *sync.WaitGroup, quit chan bool) {
	defer wg.Done()
	defer fmt.Println("WTS") //debug

	w := socket.Writer
	err := w.PrintfLine(<-srvChan)
	//send all lines in srvChan to server
	for err == nil {
		select {
		case <-quit: //exit if indicated
			return
		case str := <-srvChan:
			err = w.PrintfLine(str)
		}
	}

	//print error and exit
	if err != nil {
		errOut(err, quit)
		socket.Close()
	}
}

//take input from connection and send to console channel
func readFromServer(socket *textproto.Conn, srvChan chan string, wg *sync.WaitGroup, quit chan bool) {
	defer wg.Done()
	defer fmt.Println("RFS")

	r := socket.Reader
	line, line_err := r.ReadLine()
	for ; line_err == nil; line, line_err = r.ReadLine() {
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
	cmdRegex := regexp.MustCompile(`:(.*)?!~(.*)?@(.*)? PRIVMSG (.*) :yaircb:\s*(.*)`)

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
			} else if match := cmdRegex.FindStringSubmatch(line); match != nil {
				if cmd, valid := funcMap[match[5]]; valid {
					cmd(writeChan, match[4], match[1], "")
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
	funcMap = initMap()
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
	http.HandleFunc("/register/", registerHandler)
	http.HandleFunc("/login/", loginHandler)
	http.HandleFunc("/loginCheck/", loginCheckHandler)
	http.HandleFunc("/", registerHandler)
	http.HandleFunc("/save/", saveHandler)
	http.HandleFunc("/newUser/", newUserHandler)
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
			socket, err := textproto.Dial("tcp", "chat.freenode.net:6667")
			if err != nil {
				errOut(err, quit)
				return
			}
			//make writer/reader to/from server
			//send initial IRC messages, NICK and USER
			err = socket.Writer.PrintfLine("NICK yaircb")
			if err != nil {
				errOut(err, quit)
			}
			wgSrv.Add(1)
			//launch routine to write server output to console
			go readFromServer(socket, readChan, &wgSrv, quit)
			err = socket.Writer.PrintfLine("USER yaircb * * yaircb")
			if err != nil {
				errOut(err, quit)
			}
			//join first channel
			err = socket.Writer.PrintfLine("JOIN #ttestt")
			if err != nil {
				errOut(err, quit)
			}
			wgSrv.Add(1)
			//launch routine to send to server and get input from console
			go writeToServer(socket, writeChan, &wgSrv, quit)
			wgSrv.Wait()
		}
	}
	wg.Wait()
}

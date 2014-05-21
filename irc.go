package main

import (
	"bufio"
	"fmt"
	"net/textproto"
	"os"
	"regexp"
	"runtime"
	"sync"
)

//output err
func errOut(err error, quit chan bool) {
	fmt.Println("ERROR: ", err.Error())
	var trace []byte
	runtime.Stack(trace, false)
	fmt.Print(trace)
	if err.Error() == "EOF" {
		fmt.Println("EXITING")
		for i := 0; i < 3; i++ {
			quit <- true
		}
		fmt.Println("QUITS SENT")
	}
}

//take input from srvChan and send to server
func writeToServer(socket *textproto.Conn, srvChan chan string, wg *sync.WaitGroup, quit chan bool) {
	defer wg.Done()
	defer fmt.Println("WTS")
	w := socket.Writer
	err := w.PrintfLine(<-srvChan)
	for ; err == nil; {
		select {
		case <- quit:
			return
		case str := <- srvChan:
			err = w.PrintfLine(str)
		}
	}
	if err != nil {
		errOut(err, quit)
		socket.Close()
	}
}

//take input from connection and write out to console, also handle PING/PONG
func writeToConsole(socket *textproto.Conn, srvChan chan string, wg *sync.WaitGroup, quit chan bool) {
	defer wg.Done()
	defer fmt.Println("WTC")
	r := socket.Reader
	pingRegex := regexp.MustCompile("^PING (.*)")
	line, line_err := r.ReadLine()
	for ; line_err == nil; line, line_err = r.ReadLine() {
		select {
		case <- quit:
			return
		default:
		}
		fmt.Println(line)
		if match := pingRegex.FindStringSubmatch(line); match != nil {
			srvChan <- ("PONG " + match[1])
			fmt.Println("PONG", match[1])
		}
	}
	if line_err != nil {
		errOut(line_err, quit)
		socket.Close()
	}
}

//read input from console and send to srvChan
func readFromConsole(srvChan chan string, wg *sync.WaitGroup, quit chan bool) {
	defer wg.Done()
	defer fmt.Println("RFC")
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		select {
		case <- quit:
			return
		default:
		}
		srvChan <- in.Text()
	}
	if err := in.Err(); err != nil {
		errOut(err, quit)
	}
}

func main() {
	//funcMap := initMap()
	srvChan := make(chan string)
	//initiate connection
	socket, err := textproto.Dial("tcp", "irc.tamu.edu:6667")
	quit := make(chan bool, 3)
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
	var wg sync.WaitGroup
	wg.Add(1)
	//launch routine to write server output to console
	go writeToConsole(socket, srvChan, &wg, quit)
	err = socket.Writer.PrintfLine("USER yaircb * * yaircb")
	if err != nil {
		errOut(err, quit)
	}
	//join first channel
	err = socket.Writer.PrintfLine("JOIN #ttestt")
	if err != nil {
		errOut(err, quit)
	}
	wg.Add(2)
	//launch routine to send to server and get input from console
	go writeToServer(socket, srvChan, &wg, quit)
	go readFromConsole(srvChan, &wg, quit)
	wg.Wait()
}

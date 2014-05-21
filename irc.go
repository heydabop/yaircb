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
func errOut(err error) {
	fmt.Println("ERROR: ", err.Error())
	var trace []byte
	runtime.Stack(trace, false)
	fmt.Print(trace)
}

//take input from srvChan and send to server
func writeToServer(w textproto.Writer, srvChan chan string, wg sync.WaitGroup) {
	defer wg.Done()
	err := w.PrintfLine(<-srvChan)
	for ; err == nil; w.PrintfLine(<-srvChan) {
	}
	if err != nil {
		errOut(err)
	}
}

//take input from connection and write out to console, also handle PING/PONG
func writeToConsole(r textproto.Reader, srvChan chan string, wg sync.WaitGroup) {
	defer wg.Done()
	pingRegex := regexp.MustCompile("^PING (.*)")
	line, line_err := r.ReadLine()
	for ; line_err == nil; line, line_err = r.ReadLine() {
		fmt.Println(line)
		if match := pingRegex.FindStringSubmatch(line); match != nil {
			srvChan <- ("PONG " + match[1])
			fmt.Println("PONG", match[1])
		}
	}
	if line_err != nil {
		errOut(line_err)
	}
}

//read input from console and send to srvChan
func readFromConsole(srvChan chan string, wg sync.WaitGroup) {
	defer wg.Done()
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		srvChan <- in.Text()
	}
		errOut(err)
	if err := in.Err(); err != nil {
	}
}

func main() {
	funcMap := initMap()
	srvChan := make(chan string)
	//initiate connection
	socket, err := textproto.Dial("tcp", "irc.tamu.edu:6667")
	if err != nil {
		errOut(err)
		return
	}
	//make writer/reader to/from server
	r := socket.Reader
	w := socket.Writer
	//send initial IRC messages, NICK and USER
	err = w.PrintfLine("NICK yaircb")
	if err != nil {
		errOut(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	//launch routine to write server output to console
	go writeToConsole(r, srvChan, wg)
	err = w.PrintfLine("USER yaircb * * yaircb")
	if err != nil {
		errOut(err)
	}
	//join first channel
	err = w.PrintfLine("JOIN #ttestt")
	if err != nil {
		errOut(err)
	}
	wg.Add(2)
	//launch routine to send to server and get input from console
	go writeToServer(w, srvChan, wg)
	go readFromConsole(srvChan, wg)
	//test function map
	f, fValid := funcMap["source"]
	if fValid {
		f(srvChan, "#ttestt", "", "")
	} else {
		fmt.Println("ERROR RUNNING SOURCE")
	}
	wg.Wait()
}

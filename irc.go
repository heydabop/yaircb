package main

import(
	"fmt"
	"net/textproto"
	"time"
	"sync"
	"os"
	"bufio"
	"regexp"
	"runtime"
)

type command func(*textproto.Conn, string, string, string)

var funcMap map[string]command = make(map[string]command)

func errOut(err error){
	fmt.Println("ERROR: ", err.Error())
	var trace []byte
	runtime.Stack(trace, false)
	fmt.Print(trace)
}

func writeToServer(w textproto.Writer, srvChan chan string, wg sync.WaitGroup) {
	err := w.PrintfLine(<-srvChan)
	for ; err == nil; w.PrintfLine(<-srvChan) {}
	if err != nil {
		errOut(err)
	}
	wg.Done()
}

func writeToConsole(r textproto.Reader, srvChan chan string, wg sync.WaitGroup){
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
	wg.Done()
}

func readFromConsole(srvChan chan string, wg sync.WaitGroup){
	in := bufio.NewReader(os.Stdin)
	str, _, err := in.ReadLine()
	for ; err == nil; str, _, err = in.ReadLine() {
		srvChan <- string(str)
	}
	if err != nil {
		errOut(err)
	}
	wg.Done()
}

func main(){
	srvChan := make(chan string)
	socket, err := textproto.Dial("tcp", "irc.tamu.edu:6667")
	if err != nil {
		errOut(err)
		return
	}
	r := socket.Reader
	w := socket.Writer
	write_err := w.PrintfLine("NICK yaircb")
	if write_err != nil {
		errOut(write_err)
	}
	time.Sleep(1 * time.Second)
	var wg sync.WaitGroup
	wg.Add(1)
	go writeToConsole(r, srvChan, wg)
	write_err = w.PrintfLine("USER yaircb * * gobot")
	if write_err != nil {
		errOut(write_err)
	}
	write_err = w.PrintfLine("JOIN #ttestt")
	if write_err != nil {
		errOut(write_err)
	}
	wg.Add(2)
	go writeToServer(w, srvChan, wg)
	go readFromConsole(srvChan, wg)
	source(socket, "#ttestt", "", "")
	f := funcMap["source"]
	f(socket, "#ttestt", "", "")
	wg.Wait()
}

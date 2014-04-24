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

func readToConsole(socket *textproto.Conn, wg sync.WaitGroup){
	pingRegex := regexp.MustCompile("^PING (.*)")
	line, line_err := socket.Reader.ReadLine()
	for ; line_err == nil; line, line_err = socket.Reader.ReadLine() {
		fmt.Println(line)
		if match := pingRegex.FindStringSubmatch(line); match != nil {
			socket.Writer.PrintfLine("PONG ", match[1])
			fmt.Println("PONG", match[1])
		}
	}
	if line_err != nil {
		errOut(line_err)
	}
	wg.Done()
}

func readFromConsole(socket *textproto.Conn, wg sync.WaitGroup){
	in := bufio.NewReader(os.Stdin)
	str, _, err := in.ReadLine()
	for ; err == nil; str, _, err = in.ReadLine() {
		write_err := socket.Writer.PrintfLine(string(str))
		if write_err != nil {
			errOut(write_err)
		}
	}
	if err != nil {
		errOut(err)
	}
	wg.Done()
}

func main(){
	socket, err := textproto.Dial("tcp", "irc.tamu.edu:6667")
	if err != nil {
		errOut(err)
		return
	}
	write_err := socket.Writer.PrintfLine("NICK yaircb")
	if write_err != nil {
		errOut(write_err)
	}
	time.Sleep(1 * time.Second)
	var wg sync.WaitGroup
	wg.Add(1)
	go readToConsole(socket, wg)
	write_err = socket.Writer.PrintfLine("USER yaircb * * gobot")
	if write_err != nil {
		errOut(write_err)
	}
	write_err = socket.Writer.PrintfLine("JOIN #ttestt")
	if write_err != nil {
		errOut(write_err)
	}
	wg.Add(1)
	go readFromConsole(socket, wg)
	source(socket, "#ttestt", "", "")
	f := funcMap["source"]
	f(socket, "#ttestt", "", "")
	wg.Wait()
}

package main

import(
	"fmt"
	"net/textproto"
	"time"
	"sync"
	"os"
	"bufio"
)

func readToConsole(socket *textproto.Conn, wg sync.WaitGroup){
	line, line_err := socket.Reader.ReadLine()
	for ; line_err == nil; line, line_err = socket.Reader.ReadLine() {
		fmt.Println(line)
	}
	if line_err != nil {
		fmt.Println("LINE ERROR: ", line_err.Error())
	}
	wg.Done()
}

func readFromConsole(socket *textproto.Conn, wg sync.WaitGroup){
	in := bufio.NewReader(os.Stdin)
	str, _, err := in.ReadLine()
	for ; err == nil; str, _, err = in.ReadLine() {
		write_err := socket.Writer.PrintfLine(string(str))
		if write_err != nil {
			fmt.Println("ERROR: ", write_err.Error())
		}
	}
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
	}
	wg.Done()
}

func main(){
	socket, err := textproto.Dial("tcp", "irc.tamu.edu:6667")
	if err != nil {
		fmt.Println("ERROR: ", err.Error())
		return
	}
	write_err := socket.Writer.PrintfLine("NICK yaircb")
	if write_err != nil {
		fmt.Println("WRITE ERROR: ", write_err)
	}
	time.Sleep(3 * time.Second)
	var wg sync.WaitGroup
	wg.Add(1)
	go readToConsole(socket, wg)
	write_err = socket.Writer.PrintfLine("USER yaircb * * gobot")
	if write_err != nil {
		fmt.Println("WRITE ERROR: ", write_err)
	}
	write_err = socket.Writer.PrintfLine("JOIN #ttestt")
	if write_err != nil {
		fmt.Println("WRITE ERROR: ", write_err)
	}
	wg.Add(1)
	go readFromConsole(socket, wg)
	wg.Wait()
}

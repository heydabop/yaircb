package main

import(
	"fmt"
	"net/textproto"
	"time"
)

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
	var line string
	var line_err error
	for line, line_err = socket.Reader.ReadLine(); line_err == nil; line, line_err = socket.Reader.ReadLine() {
		fmt.Println(line)
	}
	if line_err != nil {
		fmt.Println("LINE ERROR: ", line_err.Error())
	}
}

package main

import(
	"net/textproto"
	"fmt"
)

func source(socket *textproto.Conn, channel, nick, args string){
	funcMap["source"] = command(source)
	socket.Writer.PrintfLine("PRIVMSG " + channel + " :https://github.com/heydabop/yaircb")
	fmt.Println("PRIVMSG", channel, ":https://github.com/heydabop/yaircb")
}

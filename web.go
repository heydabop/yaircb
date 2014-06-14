package main

import (
	"fmt"
	"net/http"
	"html/template"
	"github.com/fzzy/radix/redis"
)

var webDb *redis.Client

type User struct {
	Uname string
	Pwd string
}

func initWebRedis() {
	webDb, _ = redis.Dial("tcp", "127.0.0.1:6379")
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	/*p := User{}
	t, _ := template.ParseFiles("register.html")
	t.Execute(w, p)*/
	http.ServeFile(w, r, "register.html")
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	uname := r.FormValue("username")
	pwd := r.FormValue("pwd")
	fmt.Println("Form Values:", r.PostForm)
	fmt.Println("UsernameEntry:", uname)
	fmt.Println("PasswdEntry:", pwd)
	webDb.Cmd("set", uname, pwd)
		http.Redirect(w, r, "/newUser/"+uname, http.StatusFound)
}

func newUserHandler(w http.ResponseWriter, r *http.Request) {
	u := User{}
	u.Uname = r.URL.Path[len("/newUser/"):]
	reply := webDb.Cmd("get", u.Uname)
	pwd, _ := reply.Bytes()
	u.Pwd = string(pwd)
	t, err := template.ParseFiles("newUser.html")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Username:", u.Uname)
	fmt.Println("Password:", u.Pwd)
	t.Execute(w, u)
}

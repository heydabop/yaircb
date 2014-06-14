package main

import (
	"fmt"
	"github.com/fzzy/radix/redis"
	"html/template"
	"net/http"
	"time"
)

var webDb *redis.Client

type User struct {
	Uname  string
	Pwd    string
	Cookie bool
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
	webDb.Cmd("set", uname, pwd)
	expire := time.Now().AddDate(0, 0, 1)
	userCookie := http.Cookie{uname, "1234", "/", "anex.us", expire,
		expire.Format(time.UnixDate), 86400, false, false, uname + "=1234", []string{uname + "=1234"}}
	//userCookie := http.Cookie{Name: uname, Value: "1234", Expires: expire, MaxAge: 86400}
	webDb.Cmd("set", uname+"Cookie", "1234")
	webDb.Cmd("expire", uname+"Cookie", 86400)
	http.SetCookie(w, &userCookie)
	http.Redirect(w, r, "/newUser/"+uname, http.StatusFound)
}

func newUserHandler(w http.ResponseWriter, r *http.Request) {
	u := User{}
	u.Uname = r.URL.Path[len("/newUser/"):]
	reply := webDb.Cmd("get", u.Uname)
	pwd, _ := reply.Bytes()
	u.Pwd = string(pwd)
	u.Cookie = false
	t, err := template.ParseFiles("newUser.html")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Username:", u.Uname)
	fmt.Println("Password:", u.Pwd)
	cRep := webDb.Cmd("get", u.Uname+"Cookie")
	cFound, _ := cRep.Bool()
	if cFound {
		fmt.Println("found")
		c, err := r.Cookie(u.Uname)
		if err == nil {
			fmt.Println("gotCookie")
			cVal, _ := cRep.Bytes()
			if c.Value == string(cVal) {
				fmt.Println("valEqual")
				u.Cookie = true
			}
		}
	}
	t.Execute(w, u)
}

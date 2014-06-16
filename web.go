package main

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
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

func indexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "register.html")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "login.html")
}

func loginCheckHandler(w http.ResponseWriter, r *http.Request) {
	uname := r.FormValue("username")
	remember := false
	if r.FormValue("remember") == "on" {
		remember = true
	}
	pwdBytes := sha512.Sum512([]byte(r.FormValue("pwd")))
	pwd := hex.EncodeToString(pwdBytes[:])
	fmt.Println("Form Values:", r.PostForm)
	uReply := webDb.Cmd("get", uname)
	if uFound, _ := uReply.Bool(); uFound {
		fmt.Println("user found")
		if pwdDb, _ := uReply.Bytes(); pwd == string(pwdDb) {
			if remember {
				c := makeCookie(uname)
				http.SetCookie(w, &c)
			}
			fmt.Println("password match")
			t, err := template.ParseFiles("user.html")
			if err != nil {
				fmt.Println(err)
			}
			u := User{uname, pwd, remember}
			t.Execute(w, u)
		} else {
			http.Redirect(w, r, "/login/", http.StatusFound)
		}
	} else {
		http.Redirect(w, r, "/login/", http.StatusFound)
	}
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	uname := r.FormValue("username")
	pwdBytes := sha512.Sum512([]byte(r.FormValue("pwd")))
	pwd := hex.EncodeToString(pwdBytes[:])
	fmt.Println("Form Values:", r.PostForm)
	webDb.Cmd("set", uname, pwd)
	userCookie := makeCookie(uname)
	http.SetCookie(w, &userCookie)
	http.Redirect(w, r, "/user/"+uname, http.StatusFound)
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	u := User{}
	u.Uname = r.URL.Path[len("/user/"):]
	reply := webDb.Cmd("get", u.Uname)
	pwd, _ := reply.Bytes()
	u.Pwd = string(pwd)
	u.Cookie = false
	t, err := template.ParseFiles("user.html")
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
	if u.Cookie {
		t.Execute(w, u)
	} else {
		http.Redirect(w, r, "/register/", http.StatusFound)
	}
}

func makeCookie(uname string) http.Cookie {
	expire := time.Now().AddDate(0, 0, 1)
	cookieBytes := make([]byte, 64)
	rand.Read(cookieBytes)
	cookieString := hex.EncodeToString(cookieBytes)
	fmt.Println("random string:", cookieString)
	userCookie := http.Cookie{uname, cookieString, "/", "anex.us", expire, expire.Format(time.UnixDate),
		86400, true, false, uname + "=" + cookieString, []string{uname + "=" + cookieString}}
	webDb.Cmd("set", uname+"Cookie", cookieString) //this overwrites an existing cookie
	webDb.Cmd("expire", uname+"Cookie", 86400)
	return userCookie
}

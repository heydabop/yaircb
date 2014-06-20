package main

import (
	crand "crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"github.com/fzzy/radix/redis"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var webDb *redis.Client

type User struct {
	Uname  string
	Pwd    string
	Cookie bool
	Pin    string
}

func initWebRedis() {
	var err error
	webDb, err = redis.Dial("tcp", "127.0.0.1:6379")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
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
	uFound, err := uReply.Bool()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if uFound {
		fmt.Println("user found")
		pwdDb, err := uReply.Bytes()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if pwd == string(pwdDb) {
			if remember {
				c := makeCookie(uname)
				http.SetCookie(w, &c)
			}
			fmt.Println("password match")
			t, err := template.ParseFiles("user.html")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			pinReply := webDb.Cmd("get", uname+"Pin")
			pin, err := pinReply.Bytes()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			u := User{uname, pwd, remember, string(pin)}
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
	pinStr := fmt.Sprintf("%06d", rand.Intn(1000000))
	webDb.Cmd("set", uname, pwd)
	webDb.Cmd("set", uname+"Pin", pinStr)
	fmt.Println(pinStr)
	userCookie := makeCookie(uname)
	http.SetCookie(w, &userCookie)
	http.Redirect(w, r, "/user/"+uname, http.StatusFound)
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	u := User{}
	u.Uname = r.URL.Path[len("/user/"):]
	reply := webDb.Cmd("get", u.Uname)
	pwd, err := reply.Bytes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u.Pwd = string(pwd)
	reply = webDb.Cmd("get", u.Uname+"Pin")
	pin, err := reply.Bytes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u.Pin = string(pin)
	u.Cookie = false
	t, err := template.ParseFiles("user.html")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Username:", u.Uname)
	fmt.Println("Password:", u.Pwd)
	fmt.Println("Pin:", u.Pin)
	cRep := webDb.Cmd("get", u.Uname+"Cookie")
	cFound, err := cRep.Bool()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cFound {
		fmt.Println("found")
		c, err := r.Cookie(u.Uname)
		if err == nil {
			fmt.Println("gotCookie")
			cVal, err := cRep.Bytes()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
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
	crand.Read(cookieBytes)
	cookieString := hex.EncodeToString(cookieBytes)
	fmt.Println("crandom string:", cookieString)
	userCookie := http.Cookie{uname, cookieString, "/", "anex.us", expire, expire.Format(time.UnixDate),
		86400, true, false, uname + "=" + cookieString, []string{uname + "=" + cookieString}}
	webDb.Cmd("set", uname+"Cookie", cookieString) //this overwrites an existing cookie
	webDb.Cmd("expire", uname+"Cookie", 86400)
	return userCookie
}

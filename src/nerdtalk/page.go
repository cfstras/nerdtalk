package main

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Page struct {
	Title   string
	Threads []Thread
	Thread  *Thread
	Posts   []Post
	User    *User
	Date    time.Time
}

func page(w http.ResponseWriter, r *http.Request) {
	req := &Request{User: nil, W: w, R: r, State: ReqState{Unknown, ""}}
	//TODO fine-grained access
	if !req.auth() {
		w.WriteHeader(404)
		req.State.String()
		req.js(req.State)
		fmt.Fprintln(w, "\nSorry, you can't do this. Maybe you should log in.")
		return
	}

	thePage := &Page{Title: "Nerdtalk - ?", Date: time.Now(), User: req.User}
	thePage.Threads = theDB.getThreads(0, 0)
	for i := 0; i < len(thePage.Threads); i++ {
		thePage.Threads[i].Author = theDB.getUser(thePage.Threads[i].AuthorID)
	}
	if thePage.Threads != nil && len(thePage.Threads) > 0 {
		thePage.Thread = &thePage.Threads[0]
		thePage.Posts = theDB.getPosts(thePage.Thread.ID, 0, 0)
		for i := 0; i < len(thePage.Posts); i++ {
			thePage.Posts[i].Author = theDB.getUser(thePage.Posts[i].AuthorID)
		}
	}

	err := template.Must(template.ParseFiles("html/page.html")).ExecuteTemplate(w, "page.html", thePage)
	if err != nil {
		fmt.Fprintln(w, "template render failed")
		fmt.Println("Template error:", err)
	}
}

func css(w http.ResponseWriter, r *http.Request) {
	path := strings.Replace(r.URL.Path[len(URLCSS):], "..", "", -1)
	f, err := os.Open("css/" + path)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(404)
			fmt.Fprintln(w, "File not found.")
		} else if os.IsPermission(err) {
			w.WriteHeader(403)
			fmt.Fprintln(w, "Permission Denied.")
		} else {
			w.WriteHeader(500)
			fmt.Fprintln(w, "Server error.")
			fmt.Println("Error reading file:", err)
		}
		return
	}
	defer f.Close()
	w.Header().Add("Content-Type", "text/css")
	io.Copy(w, f)
}

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"net/http"
	"os"
	"strings"
	"time"
)

//constants
const URLGet = "/get/"
const URLAdd = "/add/"
const URLFiddle = "/fiddle/"

// runtime vars
var theDB *DB
var theSettings *Settings

func fiddle(w http.ResponseWriter, r *http.Request) {
	err := template.Must(template.ParseFiles("html/fiddle.html")).ExecuteTemplate(w, "fiddle.html", &Thread{})
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, "template render failed")
		fmt.Println("Tenplate error:", err)
	}
}

func api(w http.ResponseWriter, r *http.Request) {
	// user is trying to access api, he better have his passport
	if !auth(r) { //TODO add more information to auth, like user permissions
		w.WriteHeader(404)
		fmt.Fprintln(w, "Sorry, you can't do this. Maybe you should log in.")
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	switch parts[0] {
	case "get":
		get(w, r, parts[1:])
	case "add":
		add(w, r, parts[1:])

	default:
		fmt.Println("Bad Request:", r.URL, parts)
		fmt.Fprintln(w, "I'm sorry, what?")
	}
}

func checkLength(parts []string, requiredLength int, w http.ResponseWriter, r *http.Request) bool {
	if len(parts) != requiredLength {
		w.WriteHeader(404)
		fmt.Fprintln(w, "This is not the query you're looking for!\nquery:", r.URL.Path)
		return false
	}
	return true
}

func getIDCheckLength(parts []string, requiredLength int, w http.ResponseWriter, r *http.Request) (id bson.ObjectId, resume bool) {
	if resume = checkLength(parts, requiredLength, w, r); !resume {
		return
	}

	idString := parts[requiredLength-1]
	if !bson.IsObjectIdHex(idString) {
		w.WriteHeader(404)
		fmt.Println("Not an object id:", idString)
		fmt.Fprintln(w, "Invalid ID:", idString)
		resume = false
		return
	}
	id = bson.ObjectIdHex(idString)

	resume = true
	return
}

func get(w http.ResponseWriter, r *http.Request, parts []string) {
	switch parts[0] {
	case "post":
		if id, resume := getIDCheckLength(parts, 2, w, r); resume {
			js(w, theDB.getPost(id))
		}
	case "posts":
		if id, resume := getIDCheckLength(parts, 2, w, r); resume {
			js(w, theDB.getPosts(id, 0, 0))
		}
	case "threads":
		if resume := checkLength(parts, 1, w, r); resume {
			js(w, theDB.getThreads(0, 0))
		}
	case "thread":
		if id, resume := getIDCheckLength(parts, 2, w, r); resume {
			js(w, theDB.getThread(id))
		}
	case "user":
		if id, resume := getIDCheckLength(parts, 2, w, r); resume {
			js(w, theDB.getUser(id))
		}
	default:
		w.WriteHeader(404)
		fmt.Fprintln(w, "This is not the query you're looking for!\nquery:", r.URL.Path)
	}
}

func add(w http.ResponseWriter, r *http.Request, parts []string) {
	switch parts[0] {
	case "post":
		if id, resume := getIDCheckLength(parts, 2, w, r); resume {
			// add a post to thread id
			addPost(id, w, r)
		}
	case "thread":
		if resume := checkLength(parts, 1, w, r); resume {
			addThread(w, r)
		}
	case "user":
		if resume := checkLength(parts, 1, w, r); resume {
			addUser(w, r)
		}
	default:
		w.WriteHeader(404)
		fmt.Fprintln(w, "This is not the query you're looking for!\nquery:", r.URL.Path)
	}
}

func addUser(w http.ResponseWriter, r *http.Request) {
	//TODO
}

func addPost(threadID bson.ObjectId, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	textL, ok := r.Form["Text"]
	if !ok || textL == nil || len(textL) != 1 {
		w.WriteHeader(400)
		fmt.Println("No text:", r)
		fmt.Fprintln(w, "We need one Text!")
		return
	}
	text := textL[0]

	author := User{bson.NewObjectId(), "Peter", time.Now()}
	created := time.Now()
	post := &Post{ID: bson.NewObjectIdWithTime(created),
		Thread:  threadID,
		Author:  author.ID,
		Text:    text,
		Created: created}
	theDB.addPost(post)
}

func addThread(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(400)
		fmt.Println("malformed request:", r)
		fmt.Fprintln(w, "malformed request!")
		return
	}

	title := r.FormValue("Title")
	if title == "" {
		w.WriteHeader(400)
		fmt.Println("No Title:", r)
		fmt.Fprintln(w, "We need one title!")
		return
	}

	if len(title) < theSettings.Limits["thread.title.minLength"] {
		w.WriteHeader(400)
		fmt.Fprintln(w, "Title length must be at least", theSettings.Limits["thread.title.minLength"])
		return
	}

	text := r.FormValue("Text")
	if text == "" {
		w.WriteHeader(400)
		fmt.Println("No Text:", r)
		fmt.Fprintln(w, "We need one Text!")
		return
	}

	if len(text) < theSettings.Limits["post.minLength"] {
		w.WriteHeader(400)
		fmt.Fprintln(w, "Post length must be at least", theSettings.Limits["post.minLength"])
		return
	}

	author := User{bson.NewObjectId(), "Peter", time.Now()}
	created := time.Now()
	thread := &Thread{ID: bson.NewObjectIdWithTime(created),
		Title:   title,
		Author:  author.ID,
		Created: created}

	theDB.addThread(thread)

	post := &Post{ID: bson.NewObjectIdWithTime(created),
		Thread:  thread.ID,
		Author:  author.ID,
		Text:    text,
		Created: created}

	post = theDB.addPost(post)
	js(w, post)
}

func auth(r *http.Request) bool {
	return true //TODO
}

func js(w http.ResponseWriter, val interface{}) {
	ret, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		fmt.Println("JSON: ", err)
		w.WriteHeader(500)
		fmt.Fprintf(w, "Sorry, something went wrong")
		return
	}
	w.Write(ret)
}

func main() {
	//load settings //TODO
	loadSettings()
	defaults()
	saveSettings()
	//init db
	theDB = newDB("localhost", "fred", "nt", "doedel")
	defer theDB.close()
	test()

	http.HandleFunc(URLFiddle, fiddle)
	http.HandleFunc(URLGet, api)
	http.HandleFunc(URLAdd, api)
	http.ListenAndServe(":8080", nil)
}

func loadSettings() {
	s := &Settings{}

	f, err := os.Open("settings.js")
	if err != nil {
		fmt.Println("could not open settings file, using defaults")
		theSettings = &Settings{}
		return
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println("could not read settings file")
		panic(err)
	}
	json.Unmarshal(b, s)
	theSettings = s

}
func defaults() {
	s := theSettings
	if s.Limits == nil {
		s.Limits = make(map[string]int)
	}
	if s.Strings == nil {
		s.Strings = make(map[string]string)
	}

	if _, ok := s.Limits["post.minLength"]; !ok {
		s.Limits["post.minLength"] = 15
	}
	if _, ok := s.Limits["thread.title.minLength"]; !ok {
		s.Limits["thread.title.minLength"] = 5
	}
	// ...
}

func saveSettings() {
	f, err := os.Create("settings.js")
	if err != nil {
		fmt.Println("error: could not create settings file")
		return
	}
	defer f.Close()
	b, err := json.MarshalIndent(theSettings, "", "  ")
	if err != nil {
		fmt.Println("couldn't serialize settings")
		return
	}
	_, err = f.Write(b)
	if err != nil {
		fmt.Println("couldn't save settings")
		return
	}
}

func test() {

}

func debugMap(m map[string][]string) {
	fmt.Print("map[")
	for key, val := range m {
		fmt.Println(key, val, ";")
	}
	fmt.Print("]\n")
}

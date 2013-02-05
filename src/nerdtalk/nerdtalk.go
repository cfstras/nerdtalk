package main 

import (
    "net/http"
    "fmt"
    "strings"
    "strconv"
    "encoding/json"
    "time"
)

//constants
const URLGet = "/get/"
const URLAdd = "/add/"
const URLFiddle = "/fiddle/"

// runtime vars
var theDB *DB

func fiddle(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "<h1>Hello %s!</h1>", r.URL.Path[len(URLFiddle):])
}

func get(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path[len(URLGet):],"/")
	if len(parts) != 2 {
		w.WriteHeader(404)
		fmt.Fprintf(w, "This is not the query you're looking for!\nquery: ",r.URL.Path)
		return
	}
	id, err := strconv.ParseUint(parts[1], 0, 64)
	if err != nil {
		fmt.Println("Get: NaN:", r, err)
		w.WriteHeader(404)
		fmt.Fprintf(w, "This is not the query you're looking for!\nquery: ",r.URL.Path)
		return
	}
	switch parts[0] {
		case "post":
			js(w,theDB.getPost(id))
		case "posts":
			js(w,theDB.getPosts(id))
		case "thread":
			js(w,theDB.getThread(id))
		case "user":
			js(w,theDB.getUser(id))
		default:
		w.WriteHeader(404)
		fmt.Fprintf(w, "This is not the query you're looking for!\nquery: ",r.URL.Path)
	}
}

func js (w http.ResponseWriter, val interface{}) {
	ret, err := json.Marshal(val)
	if err != nil {
		fmt.Println("JSON: ",err)
		w.WriteHeader(500)
		fmt.Fprintf(w, "Sorry, something went wrong")
		return
	}
	w.Write(ret)
}

func main() {
	//init db
	theDB = newDB("localhost", "fred", "nt", "doedel")
	
	test()
	
    http.HandleFunc(URLFiddle, fiddle)
    http.HandleFunc(URLGet, get)
    
    http.ListenAndServe(":8080", nil)
    
}

func test() {
	user := &User{ID:0, Name: "Peter", Joined: time.Now()}
	theDB.users[0] = user
	thread := &Thread{ID:0, Title: "inb4 spam", Author: user, Created: time.Now()}
	theDB.threads[0] = thread 
	theDB.posts[0] = &Post{ID: 0, Thread: thread, Creator: user, Text: "OMGWTF", Created: time.Now()}
	theDB.posts[1] = &Post{ID: 1, Thread: thread, Creator: user, Text: "WEEE", Created: time.Now()}
	theDB.posts[2] = &Post{ID: 2, Thread: thread, Creator: user, Text: "THIS IS AWESOME", Created: time.Now()}
	theDB.posts[3] = &Post{ID: 3, Thread: thread, Creator: user, Text: "OH MY GOD", Created: time.Now()}
}


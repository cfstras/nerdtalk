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

func api(w http.ResponseWriter, r *http.Request) {
	// user is trying to access api, he better have his passport
	if(!auth(r)) { //TODO add more information to auth, like user permissions
		w.WriteHeader(404);
		fmt.Fprintln(w, "Sorry, you can't do this. Maybe you should log in.")
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path,"/"),"/")

	switch parts[0] {
		case "get":
			if id, resume := getIDCheckLength(parts, 3, w, r); resume {
				get(w, r, parts[1:], id)
			}
		case "add":
			if id, resume := getIDCheckLength(parts, 3, w, r); resume {
				add(w, r, parts[1:], id)
			}
		default:
			fmt.Println("Bad Request:",r.URL, parts)
			fmt.Fprintln(w, "I'm sorry, what?")
	}
}

func getIDCheckLength(parts []string, requiredLength int, w http.ResponseWriter, r *http.Request) (id uint64, resume bool) {
	if len(parts) != requiredLength {
		w.WriteHeader(404)
		fmt.Fprintln(w, "This is not the query you're looking for!\nquery:",r.URL.Path)
		resume = false
		return
	}
	id, err := strconv.ParseUint(parts[requiredLength-1], 0, 64)
	if err != nil {
		fmt.Println("Get: NaN:", r, err)
		w.WriteHeader(404)
		fmt.Fprintln(w, "This is not the query you're looking for!\nquery:",r.URL.Path)
		resume = false
		return
	}
	resume = true
	return
}

func get(w http.ResponseWriter, r *http.Request, parts []string, id uint64) {
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
		fmt.Fprintln(w, "This is not the query you're looking for!\nquery:",r.URL.Path)
	}
}

func add(w http.ResponseWriter, r *http.Request, parts []string, id uint64) {
	//TODO
}

func auth(r *http.Request) bool {
	return true //TODO
}

func js (w http.ResponseWriter, val interface{}) {
	ret, err := json.MarshalIndent(val, "", "  ")
	//TODO limit depth here
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
    http.HandleFunc(URLGet, api)
    http.HandleFunc(URLAdd, api)
    
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


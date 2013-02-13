package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strings"
	"time"
)

func fiddle(w http.ResponseWriter, r *http.Request) {
	req := &Request{User: nil, W: w, R: r, State: ReqState{Unknown, ""}}
	req.auth()
	err := template.Must(template.ParseFiles("html/fiddle.html")).ExecuteTemplate(w, "fiddle.html", req)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, "template render failed")
		fmt.Println("Tenplate error:", err)
	}
}

func api(w http.ResponseWriter, r *http.Request) {
	req := &Request{User: nil, W: w, R: r, State: ReqState{Unknown, ""}}
	// user is trying to access api, he better have his passport
	if !req.auth() {
		w.WriteHeader(404)
		req.State.String()
		req.js(req.State)
		fmt.Fprintln(w, "\nSorry, you can't do this. Maybe you should log in.")
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")

	switch parts[0] {
	case "get":
		req.get(parts[1:])
	case "add":
		req.add(parts[1:])

	default:
		fmt.Println("Bad Request:", r.URL, req, parts)
		fmt.Fprintln(w, "I'm sorry, what?")
	}
}

func (req *Request) checkLength(parts []string, requiredLength int) bool {
	if len(parts) != requiredLength {
		req.W.WriteHeader(404)
		fmt.Fprintln(req.W, "This is not the query you're looking for!\nquery:", req.R.URL.Path)
		return false
	}
	return true
}

func (req *Request) getIDCheckLength(parts []string, requiredLength int) (id bson.ObjectId, resume bool) {
	if resume = req.checkLength(parts, requiredLength); !resume {
		return
	}

	idString := parts[requiredLength-1]
	if !bson.IsObjectIdHex(idString) {
		req.W.WriteHeader(404)
		fmt.Println("Not an object id:", idString)
		fmt.Fprintln(req.W, "Invalid ID:", idString)
		resume = false
		return
	}
	id = bson.ObjectIdHex(idString)

	resume = true
	return
}

func login(w http.ResponseWriter, r *http.Request) {
	req := &Request{User: nil, W: w, R: r, State: ReqState{Unknown, ""}}

	username := r.FormValue("User")
	password := r.FormValue("Pass")
	if username != "" && password != "" {
		// User&pass auth
		user := theDB.getUserByName(username)
		if user == nil {
			req.State.AuthState = InvalidUser
		} else {
			hasher := sha256.New()
			hasher.Write([]byte(password))
			sha := hex.EncodeToString(hasher.Sum(nil))
			if sha != user.PasswordSHA {
				// wrong password.
				req.State.AuthState = WrongPassword
				fmt.Println("expected pw", user, " got", sha)
			} else {
				// yay
				req.State.AuthState = Valid
				req.User = user
			}
		}
	}

	if req.State.AuthState == Valid {
		http.SetCookie(req.W, &http.Cookie{Name: "nerdtalk-uid", Value: req.User.ID.Hex(), Expires: time.Now().AddDate(10, 0, 0)})
		http.SetCookie(req.W, &http.Cookie{Name: "nerdtalk-token", Value: req.User.AuthToken, Expires: time.Now().AddDate(10, 0, 0)})
	}

	req.State.String()
	req.js(req.State)

	//TODO OpenID

}

func (req *Request) get(parts []string) {
	switch parts[0] {
	case "post":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(theDB.getPost(id))
		}
	case "posts":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(theDB.getPosts(id, 0, 0))
		}
	case "threads":
		if resume := req.checkLength(parts, 1); resume {
			req.js(theDB.getThreads(0, 0))
		}
	case "thread":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(theDB.getThread(id))
		}
	case "user":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(theDB.getUser(id))
		}
	default:
		req.W.WriteHeader(404)
		fmt.Fprintln(req.W, "This is not the query you're looking for!\nquery:", req.R.URL.Path)
	}
}

func (req *Request) add(parts []string) {
	switch parts[0] {
	case "post":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			// add a post to thread id
			req.addPost(id)
		}
	case "thread":
		if resume := req.checkLength(parts, 1); resume {
			req.addThread()
		}
	case "user":
		if resume := req.checkLength(parts, 1); resume {
			req.addUser()
		}
	default:
		req.W.WriteHeader(404)
		fmt.Fprintln(req.W, "This is not the query you're looking for!\nquery:", req.R.URL.Path)
	}
}

func (req *Request) addUser() {
	//generate auth token
	//TODO
}

func (req *Request) addPost(threadID bson.ObjectId) {
	text := req.R.FormValue("Text")
	if text == "" {
		req.W.WriteHeader(400)
		fmt.Println("No text:", req.R)
		fmt.Fprintln(req.W, "We need a Text!")
		return
	}

	if len(text) < theSettings.Limits["post.minLength"] {
		req.W.WriteHeader(400)
		fmt.Fprintln(req.W, "Post length must be at least", theSettings.Limits["post.minLength"])
		return
	}

	created := time.Now()
	post := &Post{ID: bson.NewObjectId(),
		Thread:  threadID,
		Author:  req.User.ID,
		Text:    text,
		Created: created}
	theDB.addPost(post)
	req.js(post)
}

func (req *Request) addThread() {
	title := req.R.FormValue("Title")
	if title == "" {
		req.W.WriteHeader(400)
		fmt.Println("No Title:", req.R)
		fmt.Fprintln(req.W, "We need one title!")
		return
	}

	if len(title) < theSettings.Limits["thread.title.minLength"] {
		req.W.WriteHeader(400)
		fmt.Fprintln(req.W, "Title length must be at least", theSettings.Limits["thread.title.minLength"])
		return
	}

	text := req.R.FormValue("Text")
	if text == "" {
		req.W.WriteHeader(400)
		fmt.Println("No Text:", req.R)
		fmt.Fprintln(req.W, "We need one Text!")
		return
	}

	if len(text) < theSettings.Limits["post.minLength"] {
		req.W.WriteHeader(400)
		fmt.Fprintln(req.W, "Post length must be at least", theSettings.Limits["post.minLength"])
		return
	}

	created := time.Now()
	thread := &Thread{ID: bson.NewObjectId(),
		Title:   title,
		Author:  req.User.ID,
		Created: created}
	theDB.addThread(thread)

	post := &Post{ID: bson.NewObjectId(),
		Thread:  thread.ID,
		Author:  req.User.ID,
		Text:    text,
		Created: created}
	post = theDB.addPost(post)
	req.js(thread)
	req.js(post)
}

func (req *Request) auth() (authed bool) {
	authed = false
	req.User = nil
	uid, err := req.R.Cookie("nerdtalk-uid")
	tok, err2 := req.R.Cookie("nerdtalk-token")
	if err != nil || err2 != nil {
		req.State.AuthState = Unknown
		// there is no cookie.
		return
	}
	if !bson.IsObjectIdHex(uid.Value) {
		// too bad.
		req.State.AuthState = InvalidID
		http.SetCookie(req.W, &http.Cookie{Name: "nerdtalk-uid", Value: "", Expires: time.Now().AddDate(-1, 0, 0)})
		http.SetCookie(req.W, &http.Cookie{Name: "nerdtalk-token", Value: "", Expires: time.Now().AddDate(-1, 0, 0)})
		return
	}
	user := theDB.getUser(bson.ObjectId(uid.Value))
	if user == nil {
		//user invalid
		req.State.AuthState = InvalidUser
		http.SetCookie(req.W, &http.Cookie{Name: "nerdtalk-uid", Value: "", Expires: time.Now().AddDate(-1, 0, 0)})
		http.SetCookie(req.W, &http.Cookie{Name: "nerdtalk-token", Value: "", Expires: time.Now().AddDate(-1, 0, 0)})
		return
	}
	if user.AuthToken != tok.Value {
		//invalid auth token
		req.State.AuthState = WrongToken
		http.SetCookie(req.W, &http.Cookie{Name: "nerdtalk-token", Value: "", Expires: time.Now().AddDate(-1, 0, 0)})
		return
	}
	req.User = user
	req.State.AuthState = Valid
	authed = true
	return
}

func (req *Request) js(val interface{}) {
	var ret []byte
	var err error
	if theSettings.IndentJSON {
		ret, err = json.MarshalIndent(val, "", "  ")
	} else {
		ret, err = json.Marshal(val)
	}

	if err != nil {
		fmt.Println("JSON: ", err)
		req.W.WriteHeader(500)
		fmt.Fprintf(req.W, "Sorry, something went wrong")
		return
	}
	req.W.Write(ret)
}

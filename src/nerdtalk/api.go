package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"labix.org/v2/mgo/bson"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var safeNameReplace = regexp.MustCompile(`[^0-9A-Za-z\-]+`)

func fiddle(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
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
	fmt.Println(r)
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
	return req.getIDCheckLengthFrom(parts, requiredLength, requiredLength-1)
}

func (req *Request) getIDCheckLengthFrom(parts []string, requiredLength int, idPos int) (id bson.ObjectId, resume bool) {
	if resume = req.checkLength(parts, requiredLength); !resume {
		return
	}

	idString := parts[idPos]
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
	fmt.Println(r)
	req := &Request{User: nil, W: w, R: r, State: ReqState{Unknown, ""}}

	nickname := r.FormValue("Nick")
	password := r.FormValue("Password")
	if nickname != "" && password != "" {
		// User&pass auth
		conn := theDB.getCopy(req, "nerdtalk")
		defer conn.close()
		user := conn.getUserByNick(nickname)
		if user == nil {
			req.State.AuthState = InvalidUser
		} else {
			sha := Sha256(password)
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
	} else {
		fmt.Fprintln(req.W, "Please provide Nick and Password")
	}
	//TODO check PLogin permission!
	if req.State.AuthState == Valid {
		req.setCookies(false)
	} else if req.State.AuthState == Unknown {
		req.auth()
	}
	req.State.String()
	if req.R.FormValue("redirect") == "true" {
		http.Redirect(req.W, req.R, "/", http.StatusSeeOther)
	} else {
		req.js(req.State)
	}

	//TODO OpenID (also for logout)
	//TODO save as session in list, add session id (?)
}

func logout(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
	req := &Request{User: nil, W: w, R: r, State: ReqState{Unknown, ""}}
	authed := req.auth()
	if authed {
		newToken := RandString(32)
		conn := theDB.getCopy(req, "nerdtalk")
		defer conn.close()
		conn.setUserToken(req.User, newToken)
		//TODO check result
		// don't return the new authtoken!
	}
	// clear cookies
	req.setCookies(true)
	req.User = nil
	req.Permissions = PNone
	req.State.AuthState = Unknown
	req.State.String()
	if req.R.FormValue("redirect") == "true" {
		http.Redirect(req.W, req.R, "/", http.StatusSeeOther)
	} else {
		req.js(req.State)
	}
	
}

func (req *Request) get(parts []string) {
	conn := theDB.getCopy(req, "nerdtalk")
	defer conn.close()
	switch parts[0] {
	case "post":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(conn.getPost(id))
		}
	case "posts":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(conn.getPosts(id, 0, 0))
		}
	case "threads":
		if resume := req.checkLength(parts, 1); resume {
			req.js(conn.getThreads(0, 0))
		}
	case "thread":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(conn.getThread(id))
		}
	case "user":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(conn.getUser(id))
		}
	default:
		req.W.WriteHeader(404)
		fmt.Fprintln(req.W, "This is not the query you're looking for!\nquery:", req.R.URL.Path)
	}
}

func (req *Request) add(parts []string) {
	var ret interface{}
	var redirect string
	switch parts[0] {
	case "post":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			// add a post to thread id
			ret = req.addPost(id)
			redirect = "/thread/" + id.Hex()
		}
	case "thread":
		if resume := req.checkLength(parts, 1); resume {
			ret = req.addThread()
			redirect = "/thread/" + ret.(Thread).ID.Hex()
		}
	case "user":
		if resume := req.checkLength(parts, 1); resume {
			ret = req.addUser()
			redirect = "/user/" + ret.(User).ID.Hex()
		}
	case "like":
		if id, resume := req.getIDCheckLength(parts, 3); resume {
			ret = req.addLike(id)
			redirect = "/thread/" + parts[1]
		}
	default:
		req.W.WriteHeader(404)
		fmt.Fprintln(req.W, "This is not the query you're looking for!\nquery:", req.R.URL.Path)
		return
	}
	if req.R.FormValue("redirect") == "true" {
		//TODO if ret == nil, print on page
		http.Redirect(req.W, req.R, redirect, http.StatusSeeOther)
	} else {
		req.js(ret)
	}
}

func (req *Request) addUser() *User {
	name := req.R.FormValue("Name")
	nick := req.R.FormValue("Nick")
	pass := req.R.FormValue("Password")
	if name == "" || nick == "" || pass == "" {
		req.W.WriteHeader(400)
		fmt.Fprintln(req.W, "Please supply Name, Nick and Password!")
		return nil
	}
	user := &User{
		ID:          bson.NewObjectId(),
		Name:        name,
		Nick:        nick,
		Joined:      time.Now(),
		AuthToken:   RandString(32),
		PasswordSHA: Sha256(pass),
		Permissions: Permission(theSettings.Limits["user.default.permissions"])}
	//TODO captcha
	conn := theDB.getCopy(req, "nerdtalk")
	defer conn.close()
	user, dup := conn.addUser(user)
	if dup {
		fmt.Fprintln(req.W, "Sorry, a user with that nickname already exists.")
		return nil
	}
	if user == nil {
		fmt.Fprintln(req.W, "Sorry, an error occured.")
		return nil
	}
	return user
}

func (req *Request) addPost(threadID bson.ObjectId) *Post {
	//get thread
	conn := theDB.getCopy(req, "nerdtalk")
	defer conn.close()
	thread := conn.getThread(threadID)
	if thread == nil {
		req.W.WriteHeader(400)
		fmt.Println("Tried to post into unknown thread", req)
		fmt.Fprintln(req.W, "Sorry, I couldn't find that thread!")
		return nil
	}
	if (!thread.Internal && req.Permissions&PPost != PPost) ||
		(thread.Internal && req.Permissions&PPostInternal != PPostInternal) {
		req.W.WriteHeader(403)
		fmt.Println("User tried to post into thread he's not allowed to post!", req)
		fmt.Fprintln(req.W, "Sorry, you're not allowed to post into this thread")
		//TODO this check might be useless since the db checks this as well.
		return nil
	}

	text := req.R.FormValue("Text")
	if text == "" {
		req.W.WriteHeader(400)
		fmt.Println("No text:", req.R)
		fmt.Fprintln(req.W, "We need a Text!")
		return nil
	}

	if len(text) < theSettings.Limits["post.minLength"] {
		req.W.WriteHeader(400)
		fmt.Fprintln(req.W, "Post length must be at least", theSettings.Limits["post.minLength"])
		return nil
	}

	created := time.Now()
	post := &Post{ID: bson.NewObjectId(),
		ThreadID: threadID,
		AuthorID: req.User.ID,
		Text:     text,
		Created:  created}
	conn.addPost(post)
	return post
}

func (req *Request) addThread() bson.M {
	title := req.R.FormValue("Title")
	if title == "" {
		req.W.WriteHeader(400)
		fmt.Println("No Title:", req.R)
		fmt.Fprintln(req.W, "We need one title!")
		return nil
	}

	internalS := req.R.FormValue("Internal")
	internal := false
	if internalS != "" {
		if strings.ToLower(internalS) == "true" {
			internal = true
		} else {
			req.W.WriteHeader(400)
			fmt.Fprintf(req.W, "Bad Request.")
			return nil
		}
	}

	if (!internal && req.Permissions&PPost != PPost) ||
		(internal && req.Permissions&PPostInternal != PPostInternal) {
		req.W.WriteHeader(403)
		fmt.Println("User tried to create thread he's not allowed to post!", req)
		fmt.Fprintln(req.W, "Sorry, you're not allowed to create a thread here.")
		return nil
	}

	if len(title) < theSettings.Limits["thread.title.minLength"] {
		req.W.WriteHeader(400)
		fmt.Fprintln(req.W, "Title length must be at least", theSettings.Limits["thread.title.minLength"])
		return nil
	}

	text := req.R.FormValue("Text")
	if text == "" {
		req.W.WriteHeader(400)
		fmt.Println("No Text:", req.R)
		fmt.Fprintln(req.W, "We need one Text!")
		return nil
	}

	if len(text) < theSettings.Limits["post.minLength"] {
		req.W.WriteHeader(400)
		fmt.Fprintln(req.W, "Post length must be at least", theSettings.Limits["post.minLength"])
		return nil
	}

	safeTitle := safeNameReplace.ReplaceAllString(title, "-")
	maxl := theSettings.Limits["thread.title.safeMaxLength"]
	if len(safeTitle) > maxl {
		safeTitle = safeTitle[:maxl]
	}

	created := time.Now()
	thread := &Thread{ID: bson.NewObjectId(),
		Title:     title,
		SafeTitle: safeTitle,
		AuthorID:  req.User.ID,
		Created:   created}
	conn := theDB.getCopy(req, "nerdtalk")
	defer conn.close()
	conn.addThread(thread)

	post := &Post{ID: bson.NewObjectId(),
		ThreadID: thread.ID,
		AuthorID: req.User.ID,
		Text:     text,
		Created:  created}
	post, threadNotFound := conn.addPost(post)
	if threadNotFound {
		panic(bson.M{"Request": req, "DB": conn, "Thread": thread, "Post": post})
	}
	return bson.M{"thread": thread, "post": post}
}

func (req *Request) addLike(postID bson.ObjectId) *Like {
	if req.Permissions&PLogin != PLogin {
		fmt.Fprintln(req.W, "Sorry, you can't do this.")
		return nil
	}
	like := &Like{User: req.User.ID, Time: time.Now()}
	conn := theDB.getCopy(req, "nerdtalk")
	defer conn.close()
	like = conn.addPostLike(postID, like)
	//TODO check return
	//TODO output new post
	//TODO check auth for like
	return like
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
		req.setCookies(true)
		return
	}
	conn := theDB.getCopy(req, "nerdtalk")
	defer conn.close()
	user := conn.getUser(bson.ObjectIdHex(uid.Value))
	if user == nil || user.Name == "" {
		//user invalid
		req.State.AuthState = InvalidUser
		req.setCookies(true)
		return
	}
	//TODO use HMAC for authenticating keys (gorilla)
	if tok.Value == "" || user.AuthToken != tok.Value {
		//invalid auth token
		req.State.AuthState = WrongToken
		req.setCookies(true)
		return
	}
	req.User = user
	req.Permissions = user.Permissions
	req.State.AuthState = Valid
	authed = true
	return
}

func (req *Request) setCookies(remove bool) {
	var uid, token string
	var expire time.Time
	if remove {
		uid = ""
		token = ""
		expire = time.Now().AddDate(-1, 0, 0)
	} else {
		uid = req.User.ID.Hex()
		token = req.User.AuthToken
		expire = time.Now().AddDate(10, 0, 0)
	}
	domain := theSettings.Strings["cookies.domainName"]
	http.SetCookie(req.W, &http.Cookie{Name: "nerdtalk-uid", Value: uid, Expires: expire, Domain: domain, Path: "/"})
	http.SetCookie(req.W, &http.Cookie{Name: "nerdtalk-token", Value: token, Expires: expire, Domain: domain, Path: "/"})
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

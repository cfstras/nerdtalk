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

// Use this to get a new Request Context. Remember to `defer req.DB.Close()`
func newReq(w http.ResponseWriter, r *http.Request) *Request {
	req := &Request{User: nil, Permissions: 0, W: w, R: r, State: ReqState{Unknown, ""}}
	db := theDB.getCopy(req, "nerdtalk")
	req.DB = db
	return req
}

func api(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
	req := newReq(w, r)
	defer req.DB.close()
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
	case "del":
		req.del(parts[1:])
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
	for len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
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
	req := newReq(w, r)
	defer req.DB.close()

	nickname := r.FormValue("Nick")
	password := r.FormValue("Password")
	if nickname != "" && password != "" {
		// User&pass auth
		user := req.DB.getUserByNick(nickname, true)
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
				req.Permissions = user.Permissions
			}
		}
	} else {
		fmt.Fprintln(req.W, "Please provide Nick and Password")
	}
	//TODO check PLogin permission!
	if req.Permissions&PLogin != PLogin {
		req.W.WriteHeader(403)
		fmt.Fprintln(req.W, "Sorry, you aren't allowed to log in. Please wait for approval.")
		return
	}
	if req.State.AuthState == Valid {
		req.setCookies(false)
	} else if req.State.AuthState == Unknown {
		req.auth()
	}
	req.State.String()
	if req.R.FormValue("redirect") == "true" {
		http.Redirect(req.W, req.R, "/", http.StatusSeeOther)
		//TODO feature: use referrer to redirect appropriately
	} else {
		req.js(req.State)
	}

	//TODO OpenID (also for logout)
	//TODO save as session in list, add session id (?)
}

func logout(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
	req := newReq(w, r)
	defer req.DB.close()

	authed := req.auth()
	if authed {
		newToken := RandString(32)
		req.DB.setUserToken(req.User, newToken)
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
	switch parts[0] {
	case "post":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(req.DB.getPost(id))
		}
	case "posts":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(req.DB.getPosts(id, 0, 0))
		}
	case "threads":
		if resume := req.checkLength(parts, 1); resume {
			req.js(req.DB.getThreads(0, 0))
		}
	case "thread":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(req.DB.getThread(id))
		}
	case "user":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			req.js(req.DB.getUser(id, false))
		}
	default:
		req.W.WriteHeader(404)
		fmt.Fprintln(req.W, "This is not the query you're looking for!\nquery:", req.R.URL.Path)
	}
}

func (req *Request) add(parts []string) {
	var ret interface{} //TODO replace this with an Interface like JSONAble
	var redirect string
	switch parts[0] {
	case "post":
		if id, resume := req.getIDCheckLength(parts, 2); resume {
			// add a post to thread id
			post := req.addPost(id)
			if post != nil {
				redirect = "/thread/" + id.Hex() + "/"
				if thread := req.DB.getThread(id); thread != nil {
					redirect += thread.SafeTitle + "/"
				} else {
					redirect += "null/"
				}
				redirect += "#" + post.ID.Hex()
				ret = post
			} else {
				redirect = "/thread/" + id.Hex() + "/?err=addpost" //TODO more detailed error redirects
				lastP := req.DB.getPosts(id, -1, 1)
				if len(lastP) > 0 {
					redirect += "#" + lastP[0].ID.Hex()
				}
			}
		}
	case "thread":
		if resume := req.checkLength(parts, 1); resume {
			thread := req.addThread()
			if thread != nil {
				redirect = "/thread/" + thread.ID.Hex() + "/" + thread.SafeTitle
				ret = thread
			} else {
				redirect = "/?err=addthread"
			}
		}
	case "user":
		if resume := req.checkLength(parts, 1); resume {
			user := req.addUser()
			if user != nil {
				ret = user
				redirect = "/user/" + user.ID.Hex()
			} else {
				redirect = "/?err=adduser"
			}
		}
	case "like":
		if id, resume := req.getIDCheckLength(parts, 3); resume {
			ret = req.addLike(id)
			redirect = "/thread/" + parts[1] + "/null/#" + parts[2]
		}
	default:
		req.W.WriteHeader(404)
		fmt.Fprintln(req.W, "This is not the query you're looking for!\nquery:", req.R.URL.Path)
		return
	}
	if req.R.FormValue("redirect") == "true" {
		http.Redirect(req.W, req.R, redirect, http.StatusSeeOther)
	} else {
		req.js(ret)
	}
}

func (req *Request) del(parts []string) {
	var ret interface{} //TODO replace this with an Interface like JSONAble
	var redirect string
	switch parts[0] {
	case "like":
		if id, resume := req.getIDCheckLength(parts, 3); resume {
			ret = req.delLike(id)
			redirect = "/thread/" + parts[1] + "/null/#" + parts[2]
		}
	default:
		req.W.WriteHeader(404)
		fmt.Fprintln(req.W, "This is not the query you're looking for!\nquery:", req.R.URL.Path)
		return
	}
	if req.R.FormValue("redirect") == "true" {
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
	user, dup := req.DB.addUser(user)
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
	thread := req.DB.getThread(threadID)
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
	post, threadNotFound := req.DB.addPost(post)
	if threadNotFound {
		fmt.Fprintln(req.W, "Sorry, I couldn't find that thread!")
	}
	return post
}

func (req *Request) addThread() *Thread {
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
		fmt.Println("User tried to create thread he's not allowed to post!", "perm:", req.Permissions, "PPost", PPost, "&", req.Permissions&PPost, "Internal", internal, req)
		fmt.Fprintln(req.W, "Sorry, you're not allowed to create a thread here.")
		return nil
	}

	if len(title) < theSettings.Limits["thread.title.minLength"] {
		req.W.WriteHeader(400)
		fmt.Fprintln(req.W, "Title length must be at least", theSettings.Limits["thread.title.minLength"])
		return nil
	}

	if len(title) > theSettings.Limits["thread.title.maxLength"] {
		req.W.WriteHeader(400)
		fmt.Fprintln(req.W, "Title length can not exceed ", theSettings.Limits["thread.title.maxLength"])
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
	req.DB.addThread(thread)

	return thread
}

func (req *Request) addLike(postID bson.ObjectId) *bson.ObjectId {
	if req.Permissions&PLogin != PLogin {
		fmt.Fprintln(req.W, "Sorry, you can't do this.")
		return nil
	}
	like := req.DB.addPostLike(postID, &req.User.ID)
	//TODO check return for like
	//TODO check auth for like
	//TODO fix double-likes and make unlike functionality
	return like
}

func (req *Request) delLike(postID bson.ObjectId) *Likes {
	if req.Permissions&PLogin != PLogin {
		fmt.Fprintln(req.W, "Sorry, you can't do this.")
		return nil
	}
	likes := req.DB.delMyPostLike(postID, &req.User.ID)
	//TODO check auth for like
	//TODO fix double-likes and make unlike functionality
	return likes
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
	user := req.DB.getUser(bson.ObjectIdHex(uid.Value), true)
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

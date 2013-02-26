package main

import (
	"fmt"
	md "github.com/russross/blackfriday"
	"html/template"
	"io"
	"labix.org/v2/mgo/bson"
	"net/http"
	"os"
	"strings"
	"time"
)

type Page struct {
	Title   string
	Threads []Thread
	Thread  *Thread
	Posts   []*PagePost
	User    *User
	Date    time.Time
}

type PagePost struct {
	ID      bson.ObjectId
	Author  *User
	Created time.Time
	Likes   Likes
	ILike   bool
	Text    template.HTML
}

var theMD = md.HtmlRenderer(md.HTML_USE_SMARTYPANTS|
	md.HTML_SMARTYPANTS_LATEX_DASHES|
	md.HTML_SMARTYPANTS_FRACTIONS, "", "")

var mdExtensions int = md.EXTENSION_HARD_LINE_BREAK |
	md.EXTENSION_NO_INTRA_EMPHASIS |
	md.EXTENSION_TABLES |
	md.EXTENSION_FENCED_CODE |
	md.EXTENSION_AUTOLINK |
	md.EXTENSION_STRIKETHROUGH |
	md.EXTENSION_SPACE_HEADERS |
	md.EXTENSION_LAX_HTML_BLOCKS

func page(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r)
	req := newReq(w, r)
	defer req.DB.close()
	//TODO fine-grained access (explain?)
	req.auth()

	// find out what the user wants
	parts := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
	switch parts[0] {
	case "user":
		//TODO display info about user
		//TODO enable user approve vote
	case "thread":
		if id, resume := req.getIDCheckLengthFrom(parts, 3, 1); resume { //TODO also let 2 parts through, thread name is irrelevant anyways
			req.showThread(id)
		}
	case "favicon.ico":
		req.W.WriteHeader(404)
		return
	default:
		req.showThread("")
	}
	//TODO handle ?err=... redirects

}

func (req *Request) showThread(id bson.ObjectId) {

	thePage := &Page{Title: "nerdtalk", Date: time.Now(), User: req.User}
	conn := theDB.getCopy(req, "nerdtalk")
	defer conn.close()
	thePage.Threads = conn.getThreads(0, 0)
	for i := 0; i < len(thePage.Threads); i++ {
		thePage.Threads[i].Author = conn.getUser(thePage.Threads[i].AuthorID, false)
	}

	if id == "" {
		if thePage.Threads != nil && len(thePage.Threads) > 0 {
			thePage.Thread = &thePage.Threads[0]
		} else {
			//TODO show the rules or login help
		}
	} else {
		thePage.Thread = conn.getThread(id)
	}

	if thePage.Thread != nil {
		thePage.Title += " - " + thePage.Thread.Title
		posts := conn.getPosts(thePage.Thread.ID, 0, 0)
		thePage.Posts = make([]*PagePost, len(posts))
		for i, post := range posts {
			p := &PagePost{Text: template.HTML(md.Markdown([]byte(post.Text), theMD, mdExtensions)),
				ID:      post.ID,
				Author:  conn.getUser(post.AuthorID, false),
				Created: post.Created,
				Likes:   post.Likes}
			//TODO pretty-print date
			if req.User != nil && p.Likes.DoesUserLike(req.User.ID) {
				p.ILike = true
			}
			thePage.Posts[i] = p
			//TODO replace PagePost with a map?
		}
	}

	err := template.Must(template.ParseFiles("html/page.html")).ExecuteTemplate(req.W, "page.html", thePage)
	if err != nil {
		fmt.Fprintln(req.W, "template render failed")
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

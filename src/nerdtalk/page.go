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
	Likes   *[]Like
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
	req := &Request{User: nil, W: w, R: r, State: ReqState{Unknown, ""}}
	//TODO fine-grained access
	req.auth()

	// find out what the user wants
	parts := strings.Split(strings.TrimLeft(r.URL.Path, "/"), "/")
	switch parts[0] {
	case "user":
		//TODO display info about user
	case "thread":
		if id, resume := req.getIDCheckLengthFrom(parts, 3, 1); resume {
			req.showThread(id)
		}
	case "favicon.ico":
		req.W.WriteHeader(404)
		return
	default:
		req.showThread("")
	}

}

func (req *Request) showThread(id bson.ObjectId) {

	thePage := &Page{Title: "nerdtalk", Date: time.Now(), User: req.User}
	thePage.Threads = theDB.getThreads(0, 0)
	for i := 0; i < len(thePage.Threads); i++ {
		thePage.Threads[i].Author = theDB.getUser(thePage.Threads[i].AuthorID)
	}

	if id == "" {
		if thePage.Threads != nil && len(thePage.Threads) > 0 {
			thePage.Thread = &thePage.Threads[0]
		} else {
			//TODO?
		}
	} else {
		thePage.Thread = theDB.getThread(id)
	}

	thePage.Title += " - " + thePage.Thread.Title
	posts := theDB.getPosts(thePage.Thread.ID, 0, 0)
	thePage.Posts = make([]*PagePost, len(posts))
	for i, post := range posts {
		thePage.Posts[i] = &PagePost{Text: template.HTML(md.Markdown([]byte(post.Text), theMD, mdExtensions)),
			ID:      post.ID,
			Author:  theDB.getUser(post.AuthorID),
			Created: post.Created,
			Likes:   &post.Likes}
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

package main

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"time"
)

//TODO cache requests

// A Database Connection with a database selected.
type Conn struct {
	s   *mgo.Session
	db  *mgo.Database
	req *Request
}

// A Database Session, returned by newDB().
// Use getCopy() to read&write data
type DBSession struct {
	*mgo.Session
}

// Connects to a database, returns a Session
func newDB(host, user, database, password string) DBSession {
	//init caches
	// connect to mongo
	s, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	s.SetMode(mgo.Monotonic, true)

	return DBSession{s}
}

// Closes a Session
func (db *DBSession) close() {
	db.Close()
}

// Closes a Connection
func (conn *Conn) close() {
	conn.s.Close()
	conn.s = nil
	conn.db = nil
}

// Returns a copy of this Session usable for requests.
// Don't forget to close() the returned Connection.
func (dbc *DBSession) getCopy(req *Request, database string) *Conn {
	s := dbc.Clone()
	return &Conn{s: s, db: s.DB(database), req: req}
}

func (conn *Conn) getUser(id bson.ObjectId, overrideAuth bool) *User {
	c := conn.db.C("User")
	if !overrideAuth && conn.req.Permissions&PRead != PRead {
		return nil
	}
	user := &User{}
	err := c.Find(bson.M{"_id": id}).One(user)
	if err != nil && err != mgo.ErrNotFound {
		fmt.Println("getUser", id, ":", err)
		return &User{ID: id, Name: "Unknown User", Nick: "unknown", Joined: time.Unix(0, 0)}
	}
	return user
}

func (conn *Conn) getUserByNick(name string, overrideAuth bool) *User {
	c := conn.db.C("User")
	if !overrideAuth && conn.req.Permissions&PRead != PRead {
		return nil
	}
	user := &User{}
	err := c.Find(bson.M{"nick": name}).One(user)
	if err != nil && err != mgo.ErrNotFound {
		fmt.Println("getUserByNick", name, ":", err)
		return nil
	}
	return user
}

func (conn *Conn) getThread(id bson.ObjectId) *Thread {
	c := conn.db.C("Thread")
	if conn.req.Permissions&PRead != PRead {
		return nil
	}
	thread := &Thread{}
	err := c.Find(bson.M{"_id": id}).One(thread)
	if err != nil {
		fmt.Println("Thread", id, "not found:", err)
		return nil
	}
	if (conn.req.Permissions&PRead != PRead && !thread.Internal) ||
		(conn.req.Permissions&PReadInternal != PReadInternal && thread.Internal) {
		return nil
	}
	return thread
}

func (conn *Conn) getPost(id bson.ObjectId) *Post {
	c := conn.db.C("Post")
	post := &Post{}
	err := c.Find(bson.M{"_id": id}).One(post)
	if err != nil {
		fmt.Println("Post", id, "not found:", err)
		return nil
	}
	thread := conn.getThread(post.ThreadID)
	if thread == nil {
		// check if this post belongs to a thread and we are allowed to read it.
		return nil
	}
	return post
}

func (conn *Conn) getPosts(threadID bson.ObjectId, skip, limit int) []Post {
	c := conn.db.C("Post")
	var posts []Post
	err := c.Find(bson.M{"thread": threadID}).Sort("created").Skip(skip).Limit(limit).All(&posts)
	if err != nil {
		fmt.Println("Posts to Thread", threadID, "not found:", err)
		return nil
	}
	thread := conn.getThread(threadID)
	if thread == nil {
		// check if these posts belongs to a thread and we are allowed to read it.
		return nil
	}
	return posts
}

func (conn *Conn) getThreads(skip, limit int) []Thread {
	read := conn.req.Permissions&PRead == PRead
	readInternal := conn.req.Permissions&PReadInternal == PReadInternal
	var limits map[string]interface{}
	switch {
	case !read && readInternal:
		limits = map[string]interface{}{"internal": true}
	case read && !readInternal:
		limits = map[string]interface{}{"internal": false}
	case !read && !readInternal:
		return nil
	default:
		limits = nil
	}
	c := conn.db.C("Thread")
	var threads []Thread
	err := c.Find(limits).Sort("-created").Skip(skip).Limit(limit).All(&threads)
	if err != nil {
		fmt.Println("Thread find failed:", err)
		return nil
	}
	return threads
}

func (conn *Conn) addThread(thread *Thread) *Thread {
	if (conn.req.Permissions&PPost != PPost && !thread.Internal) ||
		(conn.req.Permissions&PPostInternal != PPostInternal && thread.Internal) {
		return nil
	}
	c := conn.db.C("Thread")
	err := c.Insert(thread)
	if err != nil {
		fmt.Println("Thread insert failed:", err)
		return nil
	}
	return thread
}

func (conn *Conn) addPost(post *Post) (ret *Post, threadNotFound bool) {
	thread := conn.getThread(post.ThreadID)
	if thread == nil {
		return nil, true
	}
	if (conn.req.Permissions&PPost != PPost && !thread.Internal) ||
		(conn.req.Permissions&PPostInternal != PPostInternal && thread.Internal) {
		return nil, false
	}
	c := conn.db.C("Post")
	err := c.Insert(post)
	if err != nil {
		fmt.Println("Post insert failed:", err)
		return nil, false
	}
	return post, false
}

// Adds a new User to the database.
// If there is already a user with the same nickname, (nil, true) is returned
// On error, (nil, false) is returned
// On success, (user, false) is returned.
func (conn *Conn) addUser(user *User) (ret *User, duplicate bool) {
	//TODO add "create user" permission
	c := conn.db.C("User")
	useralt := conn.getUserByNick(user.Nick, true)
	if useralt != nil {
		return nil, true
	}
	err := c.Insert(user)
	if err != nil {
		fmt.Println("User insert failed:", err)
		return nil, false
	}
	return ret, false
}

func (conn *Conn) setUserToken(user *User, newToken string) *User {
	c := conn.db.C("User")
	err := c.UpdateId(user.ID, bson.M{"$set": bson.M{"authtoken": newToken}})
	if err != nil {
		fmt.Println("User Token failed:", err)
		return nil
	}
	user.AuthToken = newToken
	return user
}

func (conn *Conn) addPostLike(postID bson.ObjectId, like *Like) *Like {
	// No permission checking here, since we would need to grab the thread&id first.
	// Also, the user still can't read the post when he likes it, only see the likes
	//TODO prevent unauth. User to get like list for posts he shouldn't see
	c := conn.db.C("Post")
	err := c.UpdateId(postID, bson.M{"$push": bson.M{"likes": like}})
	if err != nil {
		fmt.Println("Like failed:", err)
		return nil
	}
	return like
}

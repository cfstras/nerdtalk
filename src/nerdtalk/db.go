package main

import (
	"fmt"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
)

type DB struct {
	s    *mgo.Session
	name string
}

func newDB(host, user, database, password string) *DB {
	//init caches
	db := &DB{}
	db.name = "nerdtalk"
	// connect to mongo
	var err error
	db.s, err = mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	db.s.SetMode(mgo.Monotonic, true)
	return db
}

func (db *DB) close() {
	db.s.Close()
}

func (db *DB) getUser(id bson.ObjectId) *User {
	c := db.s.DB(db.name).C("User")
	user := &User{}
	err := c.Find(bson.M{"_id": id}).One(user)
	if err != nil {
		fmt.Println("User", id, "not found:", err)
		return nil
	}
	return user
}

func (db *DB) getThread(id bson.ObjectId) *Thread {
	c := db.s.DB(db.name).C("Thread")
	thread := &Thread{}
	err := c.Find(bson.M{"_id": id}).One(thread)
	if err != nil {
		fmt.Println("Thread", id, "not found:", err)
		return nil
	}
	return thread
}

func (db *DB) getPost(id bson.ObjectId) *Post {
	c := db.s.DB(db.name).C("Post")
	post := &Post{}
	err := c.Find(bson.M{"_id": id}).One(post)
	if err != nil {
		fmt.Println("Post", id, "not found:", err)
		return nil
	}
	return post
}

func (db *DB) getPosts(threadID bson.ObjectId, skip, limit int) []Post {
	c := db.s.DB(db.name).C("Post")
	var posts []Post
	err := c.Find(bson.M{"Thread": threadID}).Sort("Created").Skip(skip).Limit(limit).All(&posts)
	if err != nil {
		fmt.Println("Posts to Thread", threadID, "not found:", err)
		return nil
	}
	return posts
}

func (db *DB) getThreads(skip, limit int) []Thread {
	c := db.s.DB(db.name).C("Thread")
	var threads []Thread
	err := c.Find(nil).Sort("-Created").Skip(skip).Limit(limit).All(&threads)
	if err != nil {
		fmt.Println("Thread find failed:", err)
		return nil
	}
	return threads
}

func (db *DB) addThread(thread *Thread) *Thread {
	c := db.s.DB(db.name).C("Thread")
	err := c.Insert(thread)
	if err != nil {
		fmt.Println("Thread insert failed:", err)
		return nil
	}
	return thread
}

func (db *DB) addPost(post *Post) *Post {
	c := db.s.DB(db.name).C("Post")
	err := c.Insert(post)
	if err != nil {
		fmt.Println("Post insert failed:", err)
		return nil
	}
	return post
}

package main

import (

)

type DB struct {
	users map[uint64]*User
	threads map[uint64]*Thread
	posts map[uint64]*Post
}

func newDB(host, user, database, password string) *DB {
	//init caches
	db := &DB{}
	db.users = make(map[uint64]*User)
	db.threads = make(map[uint64]*Thread)
	db.posts = make(map[uint64]*Post)
	//TODO connect to SQL
	return db
}

func (db *DB) getUser(id uint64) *User {
	//TODO look in database if not cached
	return db.users[id]
}

func (db *DB) getThread(id uint64) *Thread {
	//TODO look in database if not cached
	return db.threads[id]
}

func (db *DB) getPost(id uint64) *Post {
	//TODO look in database if not cached
	return db.posts[id]
}

func (db *DB) getPosts(threadID uint64) []*Post {
	//TODO look in database
	thread := db.getThread(threadID)
	if thread == nil {
		return nil
	}
	posts := make([]*Post, 0, 8)
	for _, post := range db.posts {
		if post.Thread == thread {
			posts = append(posts, post)
		}
	}
	return posts
}

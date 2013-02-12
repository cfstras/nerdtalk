package main

import (
	"labix.org/v2/mgo/bson"
	"time"
)

type Thread struct {
	ID      bson.ObjectId "_id"
	Title   string
	Author  bson.ObjectId
	Created time.Time
}

type Post struct {
	ID      bson.ObjectId "_id"
	Thread  bson.ObjectId
	Author  bson.ObjectId
	Text    string
	Created time.Time
}

type User struct {
	ID     bson.ObjectId "_id"
	Name   string
	Joined time.Time
}

type Settings struct {
	Limits  map[string]int
	Strings map[string]string
}

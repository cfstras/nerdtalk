package main

import (
	"time"
)

type Thread struct {
	ID uint64
	Title string
	Author *User
	Created time.Time
}

type Post struct {
	ID uint64
	Thread *Thread
	Creator *User
	Text string
	Created time.Time
}

type User struct {
	ID uint64
	Name string
	Joined time.Time
}

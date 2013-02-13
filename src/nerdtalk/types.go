package main

import (
	"labix.org/v2/mgo/bson"
	"net/http"
	"time"
)

type Thread struct {
	ID      bson.ObjectId "_id,omitempty"
	Title   string        "title"
	Author  bson.ObjectId "author"
	Created time.Time     "created"
}

type Post struct {
	ID      bson.ObjectId "_id,omitempty"
	Thread  bson.ObjectId "thread"
	Author  bson.ObjectId "author"
	Text    string        "text"
	Created time.Time     "created"
	Likes   []Like        `bson:"likes"`
}

type Like struct {
	User bson.ObjectId "_id"
	Time time.Time     "time"
}

type User struct {
	ID          bson.ObjectId "_id,omitempty"
	Name        string        "name"
	Joined      time.Time     "joined"
	AuthToken   string        "authtoken"
	PasswordSHA string        "passwordsha"
}

type Request struct {
	User  *User
	W     http.ResponseWriter
	R     *http.Request
	State ReqState
}

type Settings struct {
	Limits     map[string]int
	Strings    map[string]string
	IndentJSON bool
}

type ReqState struct {
	AuthState   AuthState "-"
	StringState string    "AuthState"
}

type AuthState int

const (
	Unknown AuthState = iota
	InvalidUser
	InvalidID
	WrongPassword
	WrongToken
	Valid
)

func (a *ReqState) String() string {
	switch a.AuthState {
	case Unknown:
		a.StringState = "Unknown"
	case InvalidUser:
		a.StringState = "InvalidUser"
	case InvalidID:
		a.StringState = "InvalidID"
	case WrongPassword:
		a.StringState = "WrongPassword"
	case WrongToken:
		a.StringState = "WrongToken"
	case Valid:
		a.StringState = "Valid"
	}
	return a.StringState
}

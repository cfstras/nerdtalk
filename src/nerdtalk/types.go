package main

import (
	"labix.org/v2/mgo/bson"
	"net/http"
	"time"
)

type Thread struct {
	ID        bson.ObjectId "_id,omitempty"
	Title     string        "title"
	SafeTitle string        "safetitle"
	AuthorID  bson.ObjectId "author"
	Author    *User         `bson:"-"`
	Created   time.Time     "created"
	Internal  bool          "internal"
}

type Post struct {
	ID       bson.ObjectId "_id,omitempty"
	ThreadID bson.ObjectId "thread"
	AuthorID bson.ObjectId "author"
	Thread   *Thread       `bson:"-"`
	Author   *User         `bson:"-"`
	Text     string        "text"
	Created  time.Time     "created"
	Likes    Likes         `bson:"likes"`
}

type Likes []Like

type Like struct {
	User bson.ObjectId "_id"
	Time time.Time     "time"
}

type User struct {
	ID          bson.ObjectId "_id,omitempty"
	Name        string        "name"
	Nick        string        "nick"
	Joined      time.Time     "joined"
	AuthToken   string        "authtoken"
	PasswordSHA string        "passwordsha"
	Permissions Permission    "permissions"
}

type Request struct {
	User        *User
	Permissions Permission
	W           http.ResponseWriter
	R           *http.Request
	State       ReqState
	DB          *Conn
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

type Permission int

const (
	PLogin Permission = 1 << iota
	PRead
	PReadInternal
	PPost
	PPostInternal
	PEdit
	PEditInternal
	PEditUser
	PApprove
	PNone Permission = 0
)

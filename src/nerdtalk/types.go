package main

import (
	"labix.org/v2/mgo/bson"
	"net/http"
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
	ID          bson.ObjectId "_id"
	Name        string
	Joined      time.Time
	AuthToken   string
	PasswordSHA string
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

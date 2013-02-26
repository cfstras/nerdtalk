package main

import (
	"crypto/sha256"
	"encoding/hex"
	"labix.org/v2/mgo/bson"
	"math/rand"
	"strconv"
	"time"
)

const authRandomChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890-"

const numRandoms = 32

var randoms chan *rand.Rand

const timeFormat = `%A, %e. %B '%y`

func init() {
	randoms = make(chan *rand.Rand, 32)
	t := time.Now().Unix()
	for i := 0; i < numRandoms; i++ {
		r := rand.New(rand.NewSource(t))
		randoms <- r
		t++
	}
}

func RandString(length int) string {
	r := make([]byte, length)
	rand := <-randoms
	for i := 0; i < length; i++ {
		r[i] = authRandomChars[rand.Intn(len(authRandomChars))]
	}
	randoms <- rand
	return string(r)
}

func Sha256(s string) string {
	hasher := sha256.New()
	hasher.Write([]byte(s))
	return hex.EncodeToString(hasher.Sum(nil))
}

func (likes Likes) DoesUserLike(id bson.ObjectId) bool {
	for _, like := range likes {
		if like == Like(id) {
			return true
		}
	}
	return false
}

func prettyPrint(t time.Time) string {
	dur := time.Now().Sub(t)
	if dur < time.Minute {
		tt := int(dur.Seconds())
		if tt == 1 {
			return "1 second ago"
		} else {
			return strconv.Itoa(tt) + " seconds ago"
		}
	}
	if dur < time.Hour {
		tt := int(dur.Minutes())
		if tt == 1 {
			return "1 minute ago"
		} else {
			return strconv.Itoa(tt) + " minutes ago"
		}
	}
	if dur < (time.Hour * 48) {
		tt := int(dur.Hours())
		if tt == 1 {
			return "1 hour ago"
		} else {
			return strconv.Itoa(tt) + " hours ago"
		}
	}
	if dur < (time.Hour * 24 * 14) {
		tt := int(dur.Hours() / 24)
		if tt == 1 {
			return "1 day ago"
		} else {
			return strconv.Itoa(tt) + " days ago"
		}
	}

	return t.Format(timeFormat)
}

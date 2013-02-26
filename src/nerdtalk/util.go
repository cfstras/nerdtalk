package main

import (
	"crypto/sha256"
	"encoding/hex"
	"labix.org/v2/mgo/bson"
	"math/rand"
	"time"
)

const authRandomChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890-"

const numRandoms = 32

var randoms chan *rand.Rand

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

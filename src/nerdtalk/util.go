package main

import (
	"math/rand"
	"time"
)

const authRandomChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890-"

var random *rand.Rand

func init() {
	random = rand.New(rand.NewSource(time.Now().Unix()))
}

func RandString(length int) string {
	r := make([]byte, length)
	//TODO make this threadsafe
	for i := 0; i < length; i++ {
		r[i] = authRandomChars[random.Intn(len(authRandomChars))]
	}
	return string(r)
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

//constants
const URLGet = "/get/"
const URLAdd = "/add/"
const URLFiddle = "/fiddle/"
const URLLogin = "/login/"
const URLLogout = "/logout/"
const URLPage = "/"
const URLCSS = "/css/"

// runtime vars
var theDB DBSession
var theSettings *Settings

func main() {
	//load settings
	loadSettings()
	defaults()
	saveSettings()
	//init db
	theDB = newDB("localhost", "fred", "nt", "doedel")
	defer theDB.close()

	http.HandleFunc(URLFiddle, fiddle)
	http.HandleFunc(URLGet, api)
	http.HandleFunc(URLAdd, api)
	http.HandleFunc(URLLogin, login)
	http.HandleFunc(URLLogout, logout)
	http.HandleFunc(URLPage, page)
	http.HandleFunc(URLCSS, css)

	http.ListenAndServe(":8080", nil)
}

func loadSettings() {
	s := &Settings{}

	f, err := os.Open("settings.js")
	if err != nil {
		fmt.Println("could not open settings file, using defaults")
		theSettings = &Settings{}
		return
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println("could not read settings file")
		panic(err)
	}
	json.Unmarshal(b, s)
	theSettings = s

}
func defaults() {
	s := theSettings
	if s.Limits == nil {
		s.Limits = make(map[string]int)
	}
	if s.Strings == nil {
		s.Strings = make(map[string]string)
	}

	if _, ok := s.Limits["post.minLength"]; !ok {
		s.Limits["post.minLength"] = 15
	}
	if _, ok := s.Limits["thread.title.minLength"]; !ok {
		s.Limits["thread.title.minLength"] = 5
	}
	if _, ok := s.Limits["thread.title.MaxLength"]; !ok {
		s.Limits["thread.title.maxLength"] = 50
	}
	if _, ok := s.Limits["thread.title.safeMaxLength"]; !ok {
		s.Limits["thread.title.safeMaxLength"] = 40
	}
	if _, ok := s.Limits["user.default.permissions"]; !ok {
		s.Limits["user.default.permissions"] = int(PLogin)
	}

	if _, ok := s.Strings["cookies.domainName"]; !ok {
		s.Strings["cookies.domainName"] = ""
	}

	// ...
}

func saveSettings() {
	f, err := os.Create("settings.js")
	if err != nil {
		fmt.Println("error: could not create settings file")
		return
	}
	defer f.Close()
	b, err := json.MarshalIndent(theSettings, "", "  ")
	if err != nil {
		fmt.Println("couldn't serialize settings")
		return
	}
	_, err = f.Write(b)
	if err != nil {
		fmt.Println("couldn't save settings")
		return
	}
}

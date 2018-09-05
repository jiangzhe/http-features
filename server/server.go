package main

import (
	"net/http"
	"log"
	"fmt"
	"io/ioutil"
	"time"
	"math/rand"
)

func main() {
	rand.Seed(1)

	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("/home/jiangzhe/go/src/http-examples/static"))))
	http.HandleFunc("/requests", Requests)
	http.HandleFunc("/chunks", Chunks)
	err := http.ListenAndServe(":8080", nil)
	log.Fatal(err)
}

// function to show request information
func Requests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	writeLine(w, "-- protocol")
	writeLine(w, r.Proto)
	writeLine(w, "-- request uri")
	writeLine(w, r.RequestURI)
	writeLine(w, "-- headers")
	for k, v := range r.Header {
		writeLine(w, fmt.Sprintf("%v: %v", k, v))
	}

	if err := r.ParseForm(); err == nil {
		if len(r.Form) > 0 {
			writeLine(w, "-- form")
			for k, v := range r.Form {
				writeLine(w, fmt.Sprintf("%v: %v", k, v))
			}
		}
		if len(r.PostForm) > 0 {
			writeLine(w, "-- post form")
			for k, v := range r.PostForm {
				writeLine(w, fmt.Sprintf("%v: %v", k, v))
			}
		}
	}

	if r.Body != nil {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeLine(w, "-- body")
			w.Write(body)
		}
	}
}

var charBytes = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// function to send random-string chunked response
func Chunks(w http.ResponseWriter, r *http.Request) {
	if flusher, ok := w.(http.Flusher); ok {
		for i := 0; i < 10; i++ {
			<-time.After(1 * time.Second)
			length := rand.Intn(20)
			bytes := make([]byte, length)
			for j := 0; j < length; j++ {
				bytes[j] = charBytes[rand.Intn(len(charBytes))]
			}
			writeLine(w, string(bytes))
			flusher.Flush()
		}
		return
	}

	w.WriteHeader(http.StatusInternalServerError)
}

// helper functions

var newline = []byte("\n")

func writeLine(w http.ResponseWriter, s string) {
	w.Write([]byte(s))
	w.Write(newline)
}


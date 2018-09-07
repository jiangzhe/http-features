package main

import (
	"net/http"
	"log"
	"fmt"
	"io/ioutil"
	"time"
	"math/rand"
	"strings"
	"os"
	"crypto/md5"
	"io"
	"net"
	"sync"
	"crypto/tls"
	"strconv"
	"bytes"
	"net/url"
	"html"
)

const staticDir = "/home/jiangzhe/go/src/http-features/static"

func main() {
	// for debug
	rand.Seed(1)

	serveMux := http.NewServeMux()
	serveMux.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir(staticDir))))
	serveMux.HandleFunc("/etags/", Etags)
	serveMux.HandleFunc("/caches/", Caches)
	serveMux.HandleFunc("/redirect/", Redirect)
	serveMux.HandleFunc("/requests", Requests)
	serveMux.HandleFunc("/chunks", Chunks)
	serveMux.HandleFunc("/cookies", Cookies)
	serveMux.HandleFunc("/cookies/sub1", Cookies)
	serveMux.HandleFunc("/cookies/sub1/sub2", Cookies)

	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodConnect {
				Proxy(w, r)
			} else {
				serveMux.ServeHTTP(w, r)
			}
		}),
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}
	log.Fatal(server.ListenAndServe())
}

// function to show request information
func Requests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	writeLine(w, "-- method")
	writeLine(w, r.Method)
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
		if body, err := ioutil.ReadAll(r.Body); err == nil {
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

func Etags(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/etags/") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	filePath := staticDir + "/" + strings.TrimPrefix(r.URL.Path, "/etags/")
	file, err := os.Open(filePath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer file.Close()
	buf := make([]byte, 4096)
	n := 0
	hash := md5.New()
	for {
		n, err = file.Read(buf)
		if err != nil {
			break
		}
		if n == 0 {
			continue
		}
		rb := buf[:n]
		hash.Write(rb)
	}
	if err != nil && err != io.EOF {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fileEtag := fmt.Sprintf("\"%x\"", hash.Sum(nil))
	etag := r.Header.Get("If-None-Match")
	if len(etag) != 0 && etag == fileEtag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Etag", fileEtag)
	writeFile(w, filePath)
}

func Caches(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/caches/") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if r.URL.Path == "/caches/" {
		files, err := ioutil.ReadDir(staticDir)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		buf := &bytes.Buffer{}

		expires := r.Form.Get("expires")
		chunked := r.Form.Get("chunked")
		var qs string
		if _, err := strconv.ParseUint(expires, 10, 32); err == nil {
			qs += "&expires=" + expires
		}
		if chunked == "false" {
			qs += "&chunked=" + chunked
		}
		if len(qs) > 0 {
			qs = "?" + qs[1:]
		}

		for _, file := range files {
			fmt.Fprint(buf, "<!DOCTYPE html>\n<html><head><title>caches</title></head>\n<body>\n")
			fmt.Fprintf(buf, "<a href=\"%v\">%v</a><br/>\n", "/caches/" + url.PathEscape(file.Name()) + qs, html.EscapeString(file.Name()))
			fmt.Fprint(buf, "</body></html>\n")
		}
		w.Write(buf.Bytes())
		return
	}

	filePath := staticDir + "/" + strings.TrimPrefix(r.URL.Path, "/caches/")
	file, err := os.Open(filePath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer file.Close()

	expires := r.Form.Get("expires")
	if len(expires) > 0 {
		if _, err := strconv.ParseUint(expires, 10, 32); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%v", expires))
	}

	chunked := r.Form.Get("chunked")
	if chunked == "false" {
		bs, err := ioutil.ReadAll(file)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(bs)))
		w.Write(bs)
	} else {
		io.Copy(w, file)
	}
}

func Proxy(w http.ResponseWriter, r *http.Request) {
	h, ok := w.(http.Hijacker)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	netConn, brw, err := h.Hijack()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if brw.Reader.Buffered() > 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	proxyConn, err := net.Dial("tcp", r.URL.Host)
	if err != nil {
		// write error response to hijacked connection
		netConn.Write([]byte("HTTP/1.1 500 Internal Server Error\r\n\r\n"))
		netConn.Close()
		return
	}

	// disable original deadline
	netConn.SetDeadline(time.Time{})

	var pipeClosure sync.Once

	var pipe = func(src, dst net.Conn) {
		defer pipeClosure.Do(func() {
			src.Close()
			dst.Close()
		})

		buf := make([]byte, 8192)
		for {
			src.SetReadDeadline(time.Now().Add(10 * time.Second))
			n, err := src.Read(buf)
			if err, ok := err.(net.Error); ok && err.Timeout() {
				continue
			}
			if err != nil {
				return
			}
			b := buf[:n]
			_, err = dst.Write(b)
			if err != nil {
				return
			}
		}
	}
	go pipe(netConn, proxyConn)
	go pipe(proxyConn, netConn)

	_, err = netConn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
	if err != nil {
		log.Println(err)
		pipeClosure.Do(func() {
			netConn.Close()
			proxyConn.Close()
		})
	}
}

var redirectCodes = []string{"301", "302", "307", "308"}

func Redirect(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/redirect/") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if r.URL.Path == "/redirect/" {
		buf := new(bytes.Buffer)

		fmt.Fprint(buf, "<!DOCTYPE html>\n<html><head><title>redirects</title></head>\n<body>\n")

		buf.WriteString("<div><p>GET Redirect - qs: key1=value1</p>")
		for _, code := range redirectCodes {
			fmt.Fprintf(buf, "<a href=\"/redirect/%v?key1=value1\">Redirect %v</a><br/>\n", code, code)
		}
		buf.WriteString("</div>")

		buf.WriteString("<div><p>POST Redirect - qs: key1=value1 form: fkey1=fvalue1</p>")
		for _, code := range redirectCodes {
			fmt.Fprintf(buf, `<form method="post" action="/redirect/%v?key1=value1">`, code)
			buf.WriteString(`<input name="fkey1" value="fkey2" style="display:none"/>`)
			fmt.Fprintf(buf, `<label>Redirect %v<input type="submit" value="post"/></label>`, code)
			buf.WriteString("</form><br/>")
		}
		buf.WriteString("</div>")

		fmt.Fprint(buf, "</body></html>\n")
		w.Write(buf.Bytes())
		return
	}

	code, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/redirect/"), 10, 32)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var q string
	if len(r.URL.RawQuery) != 0 {
		q = "?" + r.URL.RawQuery
	}
	w.Header().Set("Location", "/requests" + q)
	w.WriteHeader(int(code))
}

func Cookies(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	value := r.Form.Get("value")
	domain := r.Form.Get("domain")
	path := r.Form.Get("path")
	secure := r.Form.Get("secure")
	httpOnly := r.Form.Get("httpOnly")

	if len(value) == 0 {
		w.Write([]byte("no action"))
		return
	}

	var cookie = "id=" + value
	if len(domain) > 0 {
		cookie += "; Domain=" + domain
	}
	if len(path) > 0 {
		cookie += "; path=" + path
	}
	if secure == "true" {
		cookie += "; Secure"
	}
	if httpOnly == "true" {
		cookie += "; httpOnly"
	}

	w.Header().Set("Set-Cookie", cookie)
	w.Write([]byte("cookie should be set as below:\n"))
	w.Write([]byte(cookie))
}

// helper functions

var newline = []byte("\n")

func writeLine(w http.ResponseWriter, s string) {
	w.Write([]byte(s))
	w.Write(newline)
}

func writeFile(w http.ResponseWriter, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer file.Close()
	io.Copy(w, file)
}

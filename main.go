package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"
)

var tmpl *template.Template

func main() {
	tmpl = template.Must(template.ParseFiles("index.html"))

	http.HandleFunc("/", healthCheckHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.ListenAndServe(":8080", nil)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl.Execute(w, nil)
	} else if r.Method == "POST" {
		links := extractURLs(r.FormValue("urls"))
		interval, err := time.ParseDuration(r.FormValue("interval") + "s")
		if err != nil {
			http.Error(w, "Invalid interval", http.StatusBadRequest)
			return
		}

		c := make(chan string)
		// Limit concurrent goroutines to 10
		semaphore := make(chan struct{}, 10)
		for _, link := range links {
			semaphore <- struct{}{}
			go func(link string) {
				checkLink(link, interval, c)
				<-semaphore
			}(link)
		}

		go func() {
			for l := range c {
				time.Sleep(interval)
				semaphore <- struct{}{}
				go func(link string) {
					checkLink(link, interval, c)
					<-semaphore
				}(l)
			}
		}()
	}
}

func extractURLs(urls string) []string {
	// Split URLs by whitespace (including newlines)
	fields := strings.FieldsFunc(urls, func(r rune) bool {
		return r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	var result []string
	for _, f := range fields {
		if f != "" {
			result = append(result, f)
		}
	}
	return result
}

func checkLink(link string, interval time.Duration, c chan string) {
	resp, err := http.Get(link)
	if err != nil {
		fmt.Println("Error checking or website doesn't exist:", link, "-", err)
		c <- link
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println(link, "is down (status code:", resp.StatusCode, ")")
		c <- link
		return
	}

	fmt.Println(link, "is up")
	c <- link
}

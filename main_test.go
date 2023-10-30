package main

import (
	"log"
	"net/http"
	"strings"
	"testing"
)

func testRemoteServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Request URL:", r.URL)
		dumpRequest(r, true)

		w.Header().Add("X-Response-ID", "2")
	})
	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func TestRequest(t *testing.T) {
	go testRemoteServer()

	// request
	url := "http://localhost:8080/a?b=c"
	reqBody := strings.NewReader("{\"a\": 1}")
	req, err := http.NewRequest(http.MethodPost, url, reqBody)
	if err != nil {
		t.Error(err)
	}

	// set header
	req.Header.Set(HeaderProxyAuth, "test")
	req.Header.Set(HeaderProxyTarget, "http://127.0.0.1:8081")
	req.Header.Set("X-Request-ID", "1")

	// do request
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Error(err)
	}

	dumpResponse(resp, true)
}

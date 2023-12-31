package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	HeaderProxyAuth   = "X-Proxy-Auth"
	HeaderProxyTarget = "X-Proxy-Target"
)

var (
	addr       = flag.String("addr", "127.0.0.1:8080", "-addr=127.0.0.1:8080")
	proxyAuth  = flag.String("proxy-auth", "", "-proxy-auth=KEY")
	logFile    = flag.String("log", "", "-log=FILE")
	dumpBody   = flag.Bool("dump-body", false, "-dump-body=true")
	localProxy = flag.String("local-proxy", "", "-local-proxy=http://127.0.0.1:7890")
)

func main() {
	flag.Parse()

	// Setup log
	cleanup, err := setupLog(*logFile)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	http.HandleFunc("/", handleRequest)

	log.Printf("Proxy server is running on %s\n", *addr)
	err = http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func setupLog(logFile string) (func(), error) {
	if logFile == "" {
		log.SetOutput(os.Stdout)
		return nil, nil
	}
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	log.SetOutput(f)
	return func() {
		f.Close()
	}, nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Latency
	start := time.Now()
	defer func() {
		log.Printf("Latency: %s\n", time.Since(start))
	}()

	// Dump request
	dumpRequest(r, *dumpBody)

	// Authenticate
	if !authenticate(r.Header.Get(HeaderProxyAuth)) {
		log.Println("Unauthorized")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Parse proxy target
	target, err := parseProxyTarget(r.Header.Get(HeaderProxyTarget))
	if err != nil {
		log.Printf("Failed to parse proxy target: %s\n", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	r.Host = target.Host

	// Remove headers
	r.Header.Del(HeaderProxyAuth)
	r.Header.Del(HeaderProxyTarget)

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// Set up proxy transport
	if *localProxy != "" {
		pUrl, err := url.Parse(*localProxy)
		if err != nil {
			log.Printf("Failed to parse proxy URL: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		proxy.Transport = &http.Transport{
			Proxy: http.ProxyURL(pUrl),
		}
	}

	// Dump response
	proxy.ModifyResponse = modifyResponse

	// Prepare response writer
	rw := &responseWriter{w: w}

	// Dump stream response after the connection is closed
	go dumpStreamResponse(r.Context(), rw, *dumpBody)

	// Serve the request
	proxy.ServeHTTP(rw, r)
}

func authenticate(s string) bool {
	if *proxyAuth == "" {
		return true
	}

	return s == *proxyAuth
}

func parseProxyTarget(pt string) (*url.URL, error) {
	if pt == "" {
		return nil, errors.New("target is empty")
	}
	return url.Parse(pt)
}

func dumpRequest(r *http.Request, dumpBody bool) {
	reqDump, err := httputil.DumpRequest(r, dumpBody)
	if err != nil {
		log.Printf("Failed to dump request: %s\n", err)
	} else {
		log.Printf("Request: \n%s\n", reqDump)
	}
}

func dumpResponse(resp *http.Response, dumpBody bool) {
	if resp.Header.Get("Content-Encoding") == "gzip" {
		var buf bytes.Buffer
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			log.Printf("Failed to read response: %s\n", err)
			return
		}
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Bytes()
			buf.Write(line)
		}
		resp.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
	}

	responseDump, err := httputil.DumpResponse(resp, dumpBody)
	if err != nil {
		log.Printf("Failed to dump response: %s\n", err)
	} else {
		log.Printf("Response: \n%s\n", responseDump)
	}
}

func modifyResponse(resp *http.Response) error {
	// text/event-stream or text/event-stream; charset=utf-8
	contentType := resp.Header.Get("Content-Type")
	if strings.Index(contentType, "text/event-stream") != -1 {
		return nil
	}

	dumpResponse(resp, *dumpBody)
	return nil
}

func dumpStreamResponse(ctx context.Context, rw *responseWriter, dumpBody bool) {
	<-ctx.Done()

	contentType := rw.Header().Get("Content-Type")
	if strings.Index(contentType, "text/event-stream") != -1 && dumpBody {
		var b bytes.Buffer
		err := rw.Header().Write(&b)
		if err != nil {
			log.Printf("Failed to dump response: %s\n", err)
		}
		if dumpBody {
			b.WriteByte('\n')
			b.Write(rw.buf.Bytes())
		}

		log.Printf("Response: \n%s\n", b.String())
	}
}

type responseWriter struct {
	w   http.ResponseWriter
	buf bytes.Buffer
}

func (rw *responseWriter) Header() http.Header {
	return rw.w.Header()
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	rw.buf.Write(data)
	return rw.w.Write(data)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.w.WriteHeader(statusCode)
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.w.(http.Flusher); ok {
		flusher.Flush()
	}
}

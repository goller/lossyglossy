package main

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/tylerb/graceful"
)

const version = "1.0"

// Version adds a version header to response
func Version(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Proxy-Version", version)
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// Logger is middleware that logs the request
func Logger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		log.Printf("info %s", r.URL.String())
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// downloadHandler proxies a request to https://dl.influxdata.com to download
// a file using the path argument
func downloadHandler() http.HandlerFunc {
	downloads := "https://docs.influxdata.com/influxdb/v1.2/concepts/glossary"
	docsURL, err := url.Parse(downloads)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	// Copy fragment and send to influx docs
	return func(w http.ResponseWriter, r *http.Request) {
		docsURL.Fragment = r.URL.Fragment

		hc := &http.Client{
			Transport: &http.Transport{},
		}

		res, err := hc.Get(docsURL.String())
		if err != nil {
			log.Printf("error glossary term %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(res.StatusCode)

		defer res.Body.Close()
		copyResponse(w, res.Body)
	}
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func copyResponse(dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64
	for {
		nr, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF {
			log.Printf("error during body copy: %v", rerr)
		}
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rerr != nil {
			return written, rerr
		}
	}
}

// Rss represents a status feed from AWS
type Rss struct {
	TitleItemChannel []string `xml:"channel>item>title"`
}

// LatestItem will return the latest item from an RSS feed response body
func LatestItem(response []byte) (string, error) {
	rss := Rss{}
	if err := xml.Unmarshal(response, &rss); err != nil {
		return "", err
	}
	if len(rss.TitleItemChannel) == 0 {
		return "", fmt.Errorf("No items")
	}
	return rss.TitleItemChannel[0], nil
}

// healthHandler queries AWS S3 health. If unhealthy status is 500 otherwise 204
func healthHandler() http.HandlerFunc {
	s3 := "http://status.aws.amazon.com/rss/s3-us-standard.rss"
	return func(w http.ResponseWriter, r *http.Request) {
		hc := &http.Client{
			Transport: &http.Transport{},
		}

		res, err := hc.Get(s3)
		if err != nil {
			log.Printf("error getting S3 health %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer res.Body.Close()
		octets, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Printf("error reading S3 health body")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		item, err := LatestItem(octets)
		if err != nil {
			log.Printf("error interpreting S3 status response %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if strings.Contains(item, "RESOLVED") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func server() {
	log.Printf("Starting simple influx proxy on 8080")
	mux := http.NewServeMux()
	mux.HandleFunc("/health/", healthHandler())
	mux.HandleFunc("/", downloadHandler())

	cert, err := tls.LoadX509KeyPair("testing.pem", "testing.pem")
	if err != nil {
		log.Printf("error loading cert %v", err)
		return
	}

	listener, err := tls.Listen("tcp", ":8080", &tls.Config{
		Certificates: []tls.Certificate{cert},
	})
	if err != nil {
		log.Printf("error listening on port 8080 %v", err)
		return
	}

	httpServer := &graceful.Server{Server: new(http.Server)}
	httpServer.SetKeepAlivesEnabled(true)
	httpServer.TCPKeepAlive = 5 * time.Second
	httpServer.Handler = Version(Logger(mux))
	log.Fatal(httpServer.Serve(listener))
}

func main() {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		server()
	}()
	wg.Wait()
}

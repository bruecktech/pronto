package main

import (
	"crypto/sha1"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	//"os"
	//"time"
)

func newPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:   80,
		MaxActive: 12000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", ":6379")
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}

}

var pool = newPool()
var h = sha1.New()

func getCacheKey(url string) string {
	return fmt.Sprintf("%x", h.Sum([]byte(url)))
}

func cache(w http.ResponseWriter, req *http.Request) {
	//start := time.Now()

	backendServer := "http://www.google.de"
	backendURL := backendServer + req.URL.Path
	if req.URL.RawQuery != "" {
		backendURL += "?" + req.URL.RawQuery
	}

	cacheKey := getCacheKey(backendURL)

	//log.Printf("%s Calculated cache key", time.Since(start))

	c := pool.Get()

	//log.Printf("%s Got redis connection from pool", time.Since(start))
	b, err := redis.Bytes(c.Do("GET", cacheKey))
	//log.Printf("%s Looked up cache key in redis", time.Since(start))
	if err != nil {
		////log.Printf("not cached " + backendURL + ": " + cacheKey + "\n")
		log.Printf("fetching %s", backendURL)

		resp, err := http.Get(backendURL)
		if err != nil {
			log.Fatal(err)
		}

		////log.Printf("%s Got uncached response from backend", time.Since(start))
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		ttl := 100
		//log.Printf("%s Read body", time.Since(start))
		go func() {
			c.Do("SETEX", cacheKey, ttl, b)
			c.Close()
		}()

		//log.Printf("%s Wrote body to redis", time.Since(start))

	} else {
		c.Close()
		//log.Printf("%s Ready to answer", time.Since(start))
		////log.Printf("cached " + backendURL + ": " + cacheKey + "\n")
	}

	io.WriteString(w, string(b))
	////log.Printf(trace)

}

func main() {

	http.HandleFunc("/", cache)
	http.ListenAndServe(":8080", nil)
}

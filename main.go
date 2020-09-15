package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func isValidIP(ipAddress string) bool {
	re := regexp.MustCompile(`((^\s*((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\s*$)|(^\s*((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))(%.+)?\s*$))`)
	return re.MatchString(ipAddress)
}

// refer https://stackoverflow.com/a/33881296
func noCache(h http.Handler) http.Handler {
	var epoch = time.Unix(0, 0).Format(time.RFC1123)

	var noCacheHeaders = map[string]string{
		"Expires":         epoch,
		"Cache-Control":   "no-cache, private, max-age=0",
		"Pragma":          "no-cache",
		"X-Accel-Expires": "0",
	}

	var etagHeaders = []string{
		"ETag",
		"If-Modified-Since",
		"If-Match",
		"If-None-Match",
		"If-Range",
		"If-Unmodified-Since",
	}
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Delete any ETag headers that may have been set
		for _, v := range etagHeaders {
			if r.Header.Get(v) != "" {
				r.Header.Del(v)
			}
		}

		// Set our NoCache headers
		for k, v := range noCacheHeaders {
			w.Header().Set(k, v)
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

const charset = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomString(length int) string {
	return StringWithCharset(length, charset)
}

type context struct {
	password string
	key      string
	magic    string
	tmpl     *template.Template
}

func newContext(password string, key string, loginTemplate []byte) *context {

	return &context{
		password: password,
		key:      key,
		magic:    randomString(15),
		tmpl:     template.Must(template.New("").Parse(string(loginTemplate))),
	}
}

func (c *context) authentication(h http.Handler, redirectUrl string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie(c.key); err != nil {
			http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
			return
		}

		cookie, err := r.Cookie("stamp")
		if err != nil || cookie.Value != c.magic {
			http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func (c *context) auth(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		c.tmpl.Execute(w, nil)
		return
	} else {
		if r.FormValue("password") == c.password {
			expiration := time.Now()
			expiration = expiration.Add(time.Minute * 15)
			http.SetCookie(w, &http.Cookie{Name: c.key, Value: "true", Expires: expiration})
			http.SetCookie(w, &http.Cookie{Name: "stamp", Value: c.magic, Expires: expiration})
			http.Redirect(w, r, "/", http.StatusSeeOther)
		} else {
			c.tmpl.Execute(w, map[string]string{"errMsg": "wrong password"})
			return
		}

	}
}

func main() {
	ip := flag.String("ip", "", "host addr")
	port := flag.Int("p", 3000, "port to listen")
	d := flag.String("d", ".", "the directory of static file to host")
	password := flag.String("password", "", "password")
	flag.Parse()

	// check host ip valid
	hostIP := *ip
	if hostIP != "" {
		hostIP = strings.TrimSpace(hostIP)
		if !isValidIP(hostIP) {
			fmt.Fprintf(os.Stderr, "not valid ip address\n")
			os.Exit(1)
		}
	}

	// check to serve directory exist, and setup route
	directory := *d
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "could not serve the directory(%s): %v\n", directory, err)
		os.Exit(1)
	}

	if *password != "" {
		data, err := Asset("assets/index.html")
		if err != nil {
			log.Panicf("could not found template: %v", err)
		}
		context := newContext(*password, "auth", data)
		http.HandleFunc("/auth", context.auth)
		http.Handle("/", context.authentication(noCache(http.FileServer(http.Dir(directory))), "/auth"))
	} else {

		http.Handle("/", noCache(http.FileServer(http.Dir(directory))))
	}

	// start serve
	address := fmt.Sprintf("%s:%d", hostIP, *port)
	fmt.Printf("serve directory(%s) on address %s  start...\n", directory, address)

	if err := http.ListenAndServe(address, nil); err != nil {
		fmt.Fprintf(os.Stderr, "could not listen on address %s: %v\n", address, err)
		os.Exit(1)
	}
}

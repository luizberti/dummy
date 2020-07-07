package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var port = flag.Int("port", 5000, "API's HTTP serving port")

type logwriter struct{}

func (writer logwriter) Write(bytes []byte) (int, error) {
	const tsfmt = "2006-01-02T15:04:05.999Z"
	timestamp := time.Now().UTC().Format(tsfmt)

	message := strings.SplitN(string(bytes), " ", 2)
	level := fmt.Sprintf(" [%s] ", strings.ToUpper(message[0]))

	return fmt.Fprint(os.Stderr, timestamp+level+message[1])
}

func init() {
	flag.Parse()
	log.SetFlags(0)
	log.SetOutput(new(logwriter))
}

func main() {
	// HEALTHCHECKS
	http.HandleFunc("/alive", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "", http.StatusOK)
	})
	http.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		report := ""
		status := http.StatusOK
		for _, endpoint := range flag.Args() {
			resp, err := http.Get(endpoint)
			if err != nil {
				report += fmt.Sprintf("ERR    %s\n", endpoint)
				status = http.StatusInternalServerError
			} else if resp.StatusCode != http.StatusOK {
				report += fmt.Sprintf("%03d    %s\n", resp.StatusCode, endpoint)
				status = http.StatusInternalServerError
			} else {
				report += fmt.Sprintf("200    %s\n", endpoint)
			}
		}

		http.Error(w, report, status)
	})

	// DUMMY ENDPOINTS
	http.HandleFunc("/okay", func(w http.ResponseWriter, r *http.Request) {
		defer log.Println("TRACE endpoint=/okay")
		http.Error(w, "", http.StatusOK)
	})
	http.HandleFunc("/deny", func(w http.ResponseWriter, r *http.Request) {
		defer log.Println("TRACE endpoint=/deny")
		http.Error(w, "", http.StatusBadRequest)
	})
	http.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		defer log.Println("TRACE endpoint=/fail")
		http.Error(w, "", http.StatusInternalServerError)
	})

	// UTILITY ENDPOINTS
	http.HandleFunc("/reach", func(w http.ResponseWriter, r *http.Request) {
		defer log.Printf("TRACE endpoint=/reach")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "couldn't read body", http.StatusInternalServerError)
			return
		}

		report := ""
		status := http.StatusOK
		for _, endpoint := range strings.Split(string(body), "\n") {
			before := time.Now().UnixNano()
			resp, err := http.Get(endpoint)
			delta := time.Now().UnixNano() - before
			if err != nil {
				report += "ERR    "
				status = http.StatusInternalServerError
			} else if resp.StatusCode != http.StatusOK {
				report += fmt.Sprintf("%03d    ", resp.StatusCode)
				status = http.StatusInternalServerError
			} else {
				report += "200    "
			}

			if delta < 1000000 {
				// sub-millisecond
				report += strconv.FormatInt(delta, 10) + "ns    "
			} else {
				// 1ms or more
				report += fmt.Sprintf("%.2fms    ", float64(delta)/1000000)
			}

			report += endpoint + "\n"
		}

		http.Error(w, report, status)
	})
	http.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		defer log.Printf("TRACE endpoint=/log")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "couldn't read body", http.StatusInternalServerError)
			return
		}

		for _, line := range strings.Split(string(body), "\n") {
			print := strings.TrimSpace(line)
			if print != "" {
				log.Println("DEBUG " + print)
			}
		}

		http.Error(w, "", http.StatusOK)
	})
	http.HandleFunc("/setheaders", func(w http.ResponseWriter, r *http.Request) {
		defer log.Printf("TRACE endpoint=/setheaders")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "couldn't read body", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

		for _, line := range strings.Split(string(body), "\n") {
			split := strings.SplitN(line, ": ", 2)
			if len(split) == 2 {
				w.Header().Set(split[0], split[1])
			} else {
				log.Println("WARN endpoint=/setheads msg=\"ignored line\"")
			}
		}
	})
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		defer log.Println("TRACE endpoint=/echo")
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "couldn't read body", http.StatusInternalServerError)
			return
		}

		headers := ""
		for header, value := range r.Header {
			headers += fmt.Sprintf("%s: %s\n", header, value)
		}

		http.Error(w, r.Method+" /echo\n"+headers+"\n"+string(body), http.StatusOK)
	})
	http.HandleFunc("/time", func(w http.ResponseWriter, r *http.Request) {
		defer log.Println("TRACE endpoint=/time")
		const template = "UNIX %d\nUTC  %s\nSYS  %s\n"
		now := time.Now()
		msg := fmt.Sprintf(
			template,
			now.Unix(),
			now.UTC().String(),
			now.String(),
		)
		http.Error(w, msg, http.StatusOK)
	})
	http.HandleFunc("/nano", func(w http.ResponseWriter, r *http.Request) {
		// this endpoint is not traced on logs to minimize latency
		http.Error(w, strconv.FormatInt(time.Now().UnixNano(), 10), http.StatusOK)
	})

	log.Printf("INFO msg=\"serving at :%d\"\n", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

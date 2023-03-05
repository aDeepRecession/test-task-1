package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"time"
)

type MessageBrocker interface {
	Put(string, string) error
	GetChan(string) (<-chan string, error)
	Get(string) (string, error)
}

type QueueMap struct {
	qMap    map[string]chan string
	chanCap int
}

func (qm *QueueMap) PutChan(key string) chan<- string {
	ch, ok := qm.qMap[key]
	if !ok {
		qm.qMap[key] = make(chan string, qm.chanCap)
		ch = qm.qMap[key]
	}

	return ch
}

func (qm *QueueMap) GetChan(key string) <-chan string {
	ch, ok := qm.qMap[key]
	if !ok {
		qm.qMap[key] = make(chan string, qm.chanCap)
		ch = qm.qMap[key]
	}

	return ch
}

func (qm *QueueMap) Get(key string) (string, error) {
	ch, ok := qm.qMap[key]
	if !ok {
		qm.qMap[key] = make(chan string, qm.chanCap)
		ch = qm.qMap[key]
	}

	select {
	case val := <-ch:
		return val, nil
	default:
		return "", errors.New("not found")
	}
}

var (
	errNotFound     = errors.New("not found")
	errBadArguments = errors.New("bad arguments")
)

var queueMap = QueueMap{
	map[string]chan string{},
	1000,
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handle)

	port := flag.String("port", "9090", "port for a http server")
	flag.Parse()

	server := http.Server{
		Addr:    ":" + *port,
		Handler: mux,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Printf("listening on port: %v\n", *port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		log.Fatalf("server shutdown: %v\n", err)
	}

	log.Println("server exiting")
}

func handle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		val, err := handleGET(w, r)
		if errors.Is(err, errNotFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if errors.Is(err, errBadArguments) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, val)
		return
	}

	if r.Method == http.MethodPut {
		err := handlePUT(w, r)
		if errors.Is(err, errBadArguments) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		return
	}

	w.WriteHeader(http.StatusBadRequest)
}

func handlePUT(w http.ResponseWriter, r *http.Request) error {
	queueName, val, err := parsePutArgs(r.URL)
	if err != nil {
		return errBadArguments
	}

	ch := queueMap.PutChan(queueName)
	ch <- val

	return nil
}

func parsePutArgs(url *url.URL) (string, string, error) {
	parameters := url.Query()
	if len(parameters) != 1 {
		return "", "", errors.New("number of parameters is not one")
	}

	val, ok := parameters["v"]
	if !ok || len(val) != 1 {
		return "", "", errors.New("wrong parameter")
	}

	re := regexp.MustCompile(`^/([0-9a-zA-Z]+)$`)
	matches := re.FindAllStringSubmatch(url.Path, -1)
	if len(matches) != 1 || len(matches[0]) != 2 {
		return "", "", errors.New("wrong path")
	}

	path := matches[0][1]

	return path, val[0], nil
}

func handleGET(w http.ResponseWriter, r *http.Request) (string, error) {
	queueName, timeout, err := parseGetArgs(r.URL)
	if err != nil {
		return "", errBadArguments
	}

	if timeout == 0 {
		val, err := queueMap.Get(queueName)
		if err != nil {
			return "", errNotFound
		}

		return val, nil
	}

	ch := queueMap.GetChan(queueName)
	if err != nil {
		return "", errNotFound
	}

	timer := time.NewTimer(timeout).C

	select {
	case <-timer:
		return "", errNotFound
	case val := <-ch:
		return val, nil
	}
}

func parseGetArgs(url *url.URL) (string, time.Duration, error) {
	parameters := url.Query()
	if len(parameters) > 1 {
		return "", 0, errors.New("number of parameters is more than one")
	}

	timeoutValue, ok := parameters["timeout"]
	if !ok && len(parameters) > 0 {
		return "", 0, errors.New("wrong parameter")
	}

	timeout := time.Duration(0)
	if ok {
		timeoutValueInt, err := strconv.Atoi(timeoutValue[0])
		if err != nil {
			return "", 0, errors.New("timeout parameter must be int")
		}
		timeout = time.Duration(timeoutValueInt * int(time.Second))
	}

	re := regexp.MustCompile(`^/([0-9a-zA-Z]+)$`)
	matches := re.FindAllStringSubmatch(url.Path, -1)
	if len(matches) != 1 || len(matches[0]) != 2 {
		return "", 0, errors.New("wrong path")
	}

	path := matches[0][1]

	return path, timeout, nil
}

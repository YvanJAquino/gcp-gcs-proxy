// Created with Strapit
package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	proxy "github.com/YvanJAquino/gcp-gcs-proxy/pkg/proxies/gcs"
)

var (
	HOST = os.Getenv("HOST")
	PORT = os.Getenv("PORT")
	ADDR = HOST + ":" + PORT
)

func main() {
	var p http.Handler
	var err error

	parent := context.Background()

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	signalling := make(chan os.Signal, len(signals))
	signal.Notify(signalling, signals...)

	p, err = proxy.Default(parent)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:        ADDR,
		Handler:     p,
		BaseContext: func(l net.Listener) context.Context { return parent },
	}

	go func() {
		log.Printf("Listening and serving HTTP(S) on %s", ADDR)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := <-signalling
	log.Printf("%s signal received, initiating graceful shutdown", strings.ToUpper(sig.String()))
	shutCtx, cancel := context.WithTimeout(parent, time.Second*5)
	defer cancel()
	err = server.Shutdown(shutCtx)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	log.Printf("Graceful Shutdown successful")
}

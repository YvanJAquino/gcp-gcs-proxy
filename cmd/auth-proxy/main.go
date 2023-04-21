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

	proxy "github.com/YvanJAquino/gcp-gcs-proxy/pkg/proxies/auth"
)

var (
	GCSP_PROXY_HOST  = os.Getenv("GCSP_PROXY_HOST")
	GCSP_PROXY_PORT  = os.Getenv("GCSP_PROXY_PORT")
	GCSP_ADDR        = GCSP_PROXY_HOST + ":" + GCSP_PROXY_PORT
	GCSP_TARGET_ADDR = os.Getenv("GCSP_TARGET_ADDR")
)

func main() {
	parent := context.Background()

	signals := []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	signaller := make(chan os.Signal, len(signals))
	signal.Notify(signaller, signals...)

	gcspProxy := proxy.New(parent, GCSP_TARGET_ADDR)

	server := &http.Server{
		Addr:        GCSP_ADDR,
		Handler:     gcspProxy,
		BaseContext: func(l net.Listener) context.Context { return parent },
	}

	go func() {
		log.Printf("Serving traffic from %s", GCSP_ADDR)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := <-signaller
	log.Printf("%s signal received, initiating shutdown", strings.ToUpper(sig.String()))
	shutCtx, cancel := context.WithTimeout(parent, time.Second*5)
	defer cancel()

	err := server.Shutdown(shutCtx)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	log.Println("Server successfully shutdown")
}

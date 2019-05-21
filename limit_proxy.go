package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"golang.org/x/net/context"
        "golang.org/x/time/rate"
)


func connCopy(dst net.Conn, src net.Conn, limit rate.Limit) {
	var err error
	var read int
	l := rate.NewLimiter(limit, 50)
        c, _ := context.WithCancel(context.TODO())
	bytes := make([]byte, 1024)
	for   {
		l.Wait(c)
		read, err = dst.Read(bytes)
		if err != nil {
			break
		}
		_, err = src.Write(bytes[:read])
		if err != nil {
			break
		}
	}
}

func handleConn(client net.Conn, backend string, limit rate.Limit) {
	defer client.Close()
	server, err := net.Dial("tcp", backend)
	if err != nil {
		log.Println(err)
		return
	}
	go connCopy(server, client, limit)
	connCopy(client, server, limit)
	server.Close()
}

func main() {
	var listen, backend string
	var ratelimit float64
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&listen, "l", "2233", "Listen for connections on this address.")
	flags.StringVar(&backend, "b", "1.1.1.1:53", "The address of the backend to forward to.")
	flags.Float64Var(&ratelimit, "r", 60, "network rate limit")
	flags.Parse(os.Args[1:])
	if listen == "" || backend == "" {
		fmt.Fprintln(os.Stderr, "listen and backend options required")
		os.Exit(1)
	}
	limit := rate.Limit(ratelimit)
	l, err := net.Listen("tcp", ":"+listen)
	if err != nil {
		log.Panic(err)
	}
	for {
		client, err := l.Accept()
		if err != nil {
			log.Println("accept error:", err)
			break
		}
		go handleConn(client, backend, limit)
	}
}

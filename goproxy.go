package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
)

func handleConn(client net.Conn, backend string) {
	defer client.Close()
	server, err := net.Dial("tcp", backend)
	if err != nil {
		log.Println(err)
		return
	}
	go io.Copy(server, client)
	io.Copy(client, server)
}

func main() {
	var listen, backend string
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&listen, "l", "2233", "Listen for connections on this address.")
	flags.StringVar(&backend, "b", "1.1.1.1:53", "The address of the backend to forward to.")
	flags.Parse(os.Args[1:])
	if listen == "" || backend == "" {
		fmt.Fprintln(os.Stderr, "listen and backend options required")
		os.Exit(1)
	}

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
		go handleConn(client, backend)
	}
}


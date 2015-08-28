package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func broker(dst net.Conn, src net.Conn, closed chan struct{}) {
	_, err := io.Copy(dst, src)

	if err != nil {
		log.Println("Copy error:", err.Error())
	}
	if err := src.Close(); err != nil {
		log.Println("Close error:", err.Error())
	}
	closed <- struct{}{}
}

func handle(conn net.Conn, address string) {
	defer conn.Close()

	remote, err := net.Dial("tcp", address)
	if err != nil {
		log.Println(err)
		return
	}
	defer remote.Close()

	connClosed := make(chan struct{}, 1)
	remoteClosed := make(chan struct{}, 1)

	go broker(conn, remote, remoteClosed)
	go broker(remote, conn, connClosed)

	var waitFor chan struct{}
	select {
	case <-connClosed:
		remote.Close()
		waitFor = remoteClosed
	case <-remoteClosed:
		conn.Close()
		waitFor = connClosed
	}

	<-waitFor
}

func listen(port int, dst_ip string, dst_port int) {
	address := fmt.Sprintf("%s:%d", dst_ip, port)
	if dst_port > 0 {
		address = fmt.Sprintf("%s:%d", dst_ip, dst_port)
	}

	log.Println("listening to", port, "and forwarding to", address)
	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Println("Error listening:", err.Error())
		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
			return
		}

		go handle(conn, address)
	}
}

func listen_to_port(port int, dst_ip string, dst_port int) {
	for {
		listen(port, dst_ip, dst_port)
		time.Sleep(time.Millisecond * 1000)
	}
}

func main() {
	var start_port = flag.Int("start-port", 0, "start port")
	var end_port = flag.Int("end-port", 0, "end port")
	var destination_ip = flag.String("destination-ip", "", "destination ip")
	var destination_port = flag.Int("destination-port", 0, "destination port")
	var log_file = flag.String("log", "", "log file path")
	flag.Parse()

	if *start_port <= 0 {
		fmt.Println("wrong start-port")
		return
	}

	if *end_port <= 0 {
		fmt.Println("wrong end-port")
		return
	}

	if *destination_ip == "" {
		fmt.Println("wrong destination-ip")
		return
	}

	if *log_file != "" {
		f, err := os.OpenFile(*log_file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("error opening file:", err)
		}
		defer f.Close()

		log.SetOutput(f)
	}

	for i := *start_port; i <= *end_port; i++ {
		go listen_to_port(i, *destination_ip, *destination_port)
	}

	for {
		time.Sleep(time.Millisecond * 1000)
	}
}

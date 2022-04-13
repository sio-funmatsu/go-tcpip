package main

import (
	"bufio"
	"crypto/hacktls"
	"log"
	"net"
	"os"
)

// zeroSource is an io.Reader that returns an unlimited number of zero bytes.
type zeroSource struct{}

func (zeroSource) Read(b []byte) (n int, err error) {
	for i := range b {
		b[i] = 0
	}

	return len(b), nil
}

// https://gist.github.com/denji/12b3a568f092ab951456
func main() {
	cert, err := tls.LoadX509KeyPair("./my-tls.pem", "./my-tls-key.pem")
	if err != nil {
		log.Fatal(err)
	}
	w := os.Stdout
	// https://pkg.go.dev/crypto/tls#Config
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		Rand:         zeroSource{}, // for example only; don't do this.
		MaxVersion:   tls.VersionTLS12,
		KeyLogWriter: w,
	}
	ln, err := tls.Listen("tcp", ":10443", config)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		log.Printf("Client From : %v\n", conn.RemoteAddr())
		//tlsconn := tls.Server(conn, config)
		//log.Printf("State is : %v\n", tlsconn.ConnectionState())
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		_, err := r.ReadString('\n')
		if err != nil {
			//log.Println(err)
			return
		}
	}
}

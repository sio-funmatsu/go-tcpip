package main

import (
	"fmt"
	"log"
	"syscall"
	"time"
)

// おまじない
// sudo sh -c 'echo 3 > /proc/sys/net/ipv4/tcp_retries2'
// sudo iptables -A OUTPUT -p tcp --tcp-flags RST RST -j DROP

func main() {
	dest := "13.114.40.48"
	var port uint16 = 443

	syn := TCPIP{
		DestIP:   dest,
		DestPort: port,
		TcpFlag:  "SYN",
	}
	sendfd := NewTCPSocket()
	defer syscall.Close(sendfd)
	fmt.Printf("Send SYN packet to %s\n", dest)
	ack, err := startTCPConnection(sendfd, syn)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("TCP Connection is success!!\n\n")
	time.Sleep(10 * time.Millisecond)

	//serverPacket := make(chan IPTCPTLS)

	clienthello := TCPIP{
		DestIP:    dest,
		DestPort:  port,
		TcpFlag:   "PSHACK",
		SeqNumber: ack.SeqNumber,
		AckNumber: ack.AckNumber,
		Data:      NewClientHello(),
	}
	startTLSHandshake(sendfd, clienthello)
	//serverhello, err := startTLSHandshake(sendfd, clienthello, serverPacket)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//all := <-serverPacket
	//for {
	//	<-serverPacket
	//	fmt.Println(serverPacket)
	//	//switch allpacket {
	//	//
	//	//}
	//	//break
	//}

	//fin := TCPIP{
	//	DestIP:    dest,
	//	DestPort:  port,
	//	TcpFlag:   "FINACK",
	//	SeqNumber: all.TCPHeader.SequenceNumber,
	//	AckNumber: all.TCPHeader.AcknowlegeNumber,
	//}
	//fmt.Printf("Send FINACK packet to %s\n", dest)
	//_, err = startTCPConnection(sendfd, fin)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//fmt.Printf("TCP Connection Close is success!!\n")
}

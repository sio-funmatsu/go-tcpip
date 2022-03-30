package main

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"log"
	"syscall"
	"time"
)

// おまじない
// sudo sh -c 'echo 3 > /proc/sys/net/ipv4/tcp_retries2'
// sudo iptables -A OUTPUT -p tcp --tcp-flags RST RST -j DROP

func main() {
	premaster := randomByte(48)
	clientRandom := randomByte(32)
	serverRandom := randomByte(32)

	var clientServerRandomByte []byte
	clientServerRandomByte = append(clientServerRandomByte, clientRandom...)
	clientServerRandomByte = append(clientServerRandomByte, serverRandom...)

	result := createMasterSecret(premaster, clientServerRandomByte)
	result2 := createMasterSecret(premaster, clientServerRandomByte)
	fmt.Printf("%x\n", result)
	fmt.Printf("%x\n", result2)

}

func __main() {
	serverTLS, protoBytes := unpackTLSPacket(rsaByte)
	for _, v := range serverTLS {
		switch proto := v.HandshakeProtocol.(type) {
		case ServerHello:
			fmt.Printf("Cipher Suite is : %s\n", tls.CipherSuiteName(binary.BigEndian.Uint16(proto.CipherSuites)))
		case ServerCertifiate:
			fmt.Printf("%T\n", proto.Certificates[0].PublicKey)
			fmt.Println(proto.Certificates[0].PublicKey)
		case ServerHelloDone:
			fmt.Println("ServerHelloDone")
		}
	}
	_ = protoBytes
}

func _main() {
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

	var hello ClientHello

	clienthello := TCPIP{
		DestIP:    dest,
		DestPort:  port,
		TcpFlag:   "PSHACK",
		SeqNumber: ack.SeqNumber,
		AckNumber: ack.AckNumber,
		Data:      hello.NewClientHello(),
	}
	//startTLSHandshake(sendfd, clienthello)
	serverhello, err := startTLSHandshake(sendfd, clienthello)
	if err != nil {
		log.Fatal(err)
	}
	//all := <-serverPacket
	//for {
	//	<-serverPacket
	//	fmt.Println(serverPacket)
	//	//switch allpacket {
	//	//
	//	//}
	//	//break
	//}

	fin := TCPIP{
		DestIP:    dest,
		DestPort:  port,
		TcpFlag:   "FINACK",
		SeqNumber: serverhello.SequenceNumber,
		AckNumber: serverhello.AcknowlegeNumber,
	}
	fmt.Printf("Send FINACK packet to %s\n", dest)
	_, err = startTCPConnection(sendfd, fin)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("TCP Connection Close is success!!\n")
}

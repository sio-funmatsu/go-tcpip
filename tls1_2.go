package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"log"
	"syscall"
	"time"
)

func NewTLSRecordHeader(ctype string, length uint16) []byte {
	var b []byte
	switch ctype {
	case "Handshake":
		b = append(b, ContentTypeHandShake)
	case "AppDada":
		b = append(b, ContentTypeApplicationData)
	case "Alert":
		b = append(b, ContentTypeAlert)
	case "ChangeCipherSpec":
		b = append(b, HandshakeTypeChangeCipherSpec)
	}

	b = append(b, TLS1_2...)
	b = append(b, uintTo2byte(length)...)
	return b
}

func (*ClientHello) NewRSAClientHello() (clientrandom, clienthello []byte) {

	handshake := ClientHello{
		HandshakeType: []byte{HandshakeTypeClientHello},
		Length:        []byte{0x00, 0x00, 0x00},
		Version:       TLS1_2,
		Random:        noRandomByte(32),
		//SessionID:          []byte{0x00},
		SessionIDLength:    []byte{0x20},
		SessionID:          noRandomByte(32),
		CipherSuitesLength: []byte{0x00, 0x02},
		// TLS_RSA_WITH_AES_128_GCM_SHA256
		CipherSuites:      []byte{0x00, 0x9c},
		CompressionLength: []byte{0x01},
		CompressionMethod: []byte{0x00},
		Extensions:        setTLSExtenstions(),
	}

	// Typeの1byteとLengthの3byteを合計から引く
	handshake.Length = uintTo3byte(uint32(toByteLen(handshake) - 4))

	var hello []byte
	hello = append(hello, NewTLSRecordHeader("Handshake", toByteLen(handshake))...)
	hello = append(hello, toByteArr(handshake)...)

	return handshake.Random, hello
}

// GoでデフォルトでセットされるのTLS extensionを返す
func setTLSExtenstions() []byte {
	var tlsExtension []byte

	// set length
	tlsExtension = append(tlsExtension, []byte{0x00, 0x4b}...)

	//　status_reqeust
	tlsExtension = append(tlsExtension, []byte{0x00, 0x05, 0x00, 0x05, 0x01, 0x00, 0x00, 0x00, 0x00}...)
	// supported_groups
	tlsExtension = append(tlsExtension, []byte{
		0x00, 0x0a, 0x00, 0x0a, 0x00, 0x08, 0x00, 0x1d,
		0x00, 0x17, 0x00, 0x18, 0x00, 0x19,
	}...)
	// ec_point_formats
	tlsExtension = append(tlsExtension, []byte{0x00, 0x0b, 0x00, 0x02, 0x01, 0x00}...)
	// signature_algorithms
	tlsExtension = append(tlsExtension, []byte{
		0x00, 0x0d, 0x00, 0x1a, 0x00, 0x18, 0x08, 0x04,
		0x04, 0x03, 0x08, 0x07, 0x08, 0x05, 0x08, 0x06,
		0x04, 0x01, 0x05, 0x01, 0x06, 0x01, 0x05, 0x03,
		0x06, 0x03, 0x02, 0x01, 0x02, 0x03,
	}...)
	// renagotiation_info
	tlsExtension = append(tlsExtension, []byte{0xff, 0x01, 0x00, 0x01, 0x00}...)
	// signed_certificate_timestamp
	tlsExtension = append(tlsExtension, []byte{0x00, 0x12, 0x00, 0x00}...)
	// supported_versions
	tlsExtension = append(tlsExtension, []byte{0x00, 0x2b, 0x00, 0x03, 0x02, 0x03, 0x03}...)

	return tlsExtension
}

func (*ClientKeyExchange) NewClientKeyExchange(pubkey *rsa.PublicKey) (clientKeyExchange, premasterByte []byte) {

	// 46byteのランダムなpremaster secretを生成する
	// https://www.ipa.go.jp/security/rfc/RFC5246-07JA.html#07471
	//premaster := randomByte(46)
	premaster := noRandomByte(46)
	premasterByte = append(premasterByte, TLS1_2...)
	premasterByte = append(premasterByte, premaster...)

	//サーバの公開鍵で暗号化する
	secret, err := rsa.EncryptPKCS1v15(rand.Reader, pubkey, premasterByte)
	if err != nil {
		log.Fatalf("create premaster secret err : %v\n", err)
	}

	clientKey := ClientKeyExchange{
		HandshakeType:                  []byte{HandshakeTypeClientKeyExchange},
		Length:                         []byte{0x00, 0x00, 0x00},
		EncryptedPreMasterSecretLength: uintTo2byte(uint16(len(secret))),
		EncryptedPreMasterSecret:       secret,
	}

	// Lengthをセット
	clientKey.Length = uintTo3byte(uint32(toByteLen(clientKey) - 4))

	// byte配列にする
	clientKeyExchange = append(clientKeyExchange, NewTLSRecordHeader("Handshake", toByteLen(clientKey))...)
	clientKeyExchange = append(clientKeyExchange, toByteArr(clientKey)...)

	return clientKeyExchange, premasterByte
}

func NewChangeCipherSpec() []byte {
	// Type: handshake, TLS1.2のVersion, length: 2byte, message: 1byte
	return []byte{0x14, 0x03, 0x03, 0x00, 0x01, 0x01}
}

func readCertificates(packet []byte) []*x509.Certificate {

	var b []byte
	var certificates []*x509.Certificate

	//　https://pkg.go.dev/crypto/x509#SystemCertPool
	// OSにインストールされている証明書を読み込む
	ospool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("get SystemCertPool err : %v\n", err)
	}

	// TLS Handshak protocolのCertificatesのLengthが0になるまでx509証明書をReadする
	// 読み込んだx509証明書を配列に入れる
	for {
		if len(packet) == 0 {
			break
		} else {
			length := sum3BytetoLength(packet[0:3])
			//b := make([]byte, length)
			b = readByteNum(packet, 3, int64(length))
			cert, err := x509.ParseCertificate(b)
			if err != nil {
				log.Fatalf("ParseCertificate error : %s", err)
			}
			certificates = append(certificates, cert)
			//byte配列を縮める
			packet = packet[3+length:]
		}
	}

	// 証明書を検証する
	// 配列にはサーバ証明書、中間証明書の順番で格納されているので中間証明書から検証していくので
	// forloopをdecrementで回す
	for i := len(certificates) - 1; i >= 0; i-- {
		var opts x509.VerifyOptions
		if len(certificates[i].DNSNames) == 0 {
			opts = x509.VerifyOptions{
				Roots: ospool,
			}
		} else {
			opts = x509.VerifyOptions{
				DNSName: certificates[i].DNSNames[0],
				Roots:   ospool,
			}
		}

		// 検証
		_, err = certificates[i].Verify(opts)
		if err != nil {
			log.Fatalf("failed to verify certificate : %v\n", err)
		}
		if 0 < i {
			ospool.AddCert(certificates[1])
		}
	}
	fmt.Println("証明書マジ正しい！")
	return certificates
}

func unpackECDiffieHellmanParam(packet []byte) ECDiffieHellmanParam {
	return ECDiffieHellmanParam{
		CurveType:          packet[0:1],
		NamedCurve:         packet[1:3],
		PubkeyLength:       packet[3:4],
		Pubkey:             readByteNum(packet, 4, 32),
		SignatureAlgorithm: packet[36:38],
		SignatureLength:    packet[38:40],
		Signature:          packet[40:],
	}
}

func unpackTLSHandshake(packet []byte) interface{} {
	var i interface{}

	switch packet[0] {
	case HandshakeTypeServerHello:
		i = ServerHello{
			HandshakeType:     packet[0:1],
			Length:            packet[1:4],
			Version:           packet[4:6],
			Random:            packet[6:38],
			SessionID:         packet[38:39],
			CipherSuites:      packet[39:41],
			CompressionMethod: packet[41:42],
		}
		fmt.Printf("ServerHello : %+v\n", i)
		//fmt.Printf("Cipher Suite is : %s\n", tls.CipherSuiteName(binary.BigEndian.Uint16(packet[39:41])))
	case HandshakeTypeServerCertificate:
		i = ServerCertificate{
			HandshakeType:      packet[0:1],
			Length:             packet[1:4],
			CertificatesLength: packet[4:7],
			Certificates:       readCertificates(packet[7:]),
		}
		fmt.Printf("Certificate : %+v\n", i)
	case HandshakeTypeServerKeyExchange:
		i = ServerKeyExchange{
			HandshakeType:               packet[0:1],
			Length:                      packet[1:4],
			ECDiffieHellmanServerParams: unpackECDiffieHellmanParam(packet[4:]),
		}
		fmt.Printf("ServerKeyExchange : %+v\n", i)
	case HandshakeTypeServerHelloDone:
		i = ServerHelloDone{
			HandshakeType: packet[0:1],
			Length:        packet[1:4],
		}
		fmt.Printf("ServerHelloDone : %+v\n", i)
	case HandshakeTypeFinished:
	}

	return i
}

func unpackTLSPacket(packet []byte) ([]TLSProtocol, []byte) {
	var protocols []TLSProtocol
	var protocolsByte []byte

	// TCPのデータをContentType、TLSバージョンのbyte配列でSplitする
	splitByte := bytes.Split(packet, []byte{0x16, 0x03, 0x03})
	for _, v := range splitByte {
		if len(v) != 0 && (len(v)-2) == int(binary.BigEndian.Uint16(v[0:2])) {
			//fmt.Printf("%d : %d\n", len(v), binary.BigEndian.Uint16(v[0:2]))
			rHeader := TLSRecordHeader{
				ContentType:     []byte{0x16},
				ProtocolVersion: []byte{0x03, 0x03},
				Length:          v[0:2],
			}
			tls := unpackTLSHandshake(v[2:])
			proto := TLSProtocol{
				RHeader:           rHeader,
				HandshakeProtocol: tls,
			}
			protocolsByte = append(protocolsByte, v[2:]...)
			protocols = append(protocols, proto)
		} else if len(v) != 0 && bytes.Contains(v, []byte{0x00, 0x04, 0x0e}) {
			rHeader := TLSRecordHeader{
				ContentType:     []byte{0x16},
				ProtocolVersion: []byte{0x03, 0x03},
				Length:          v[0:2],
			}
			//ServerHelloDoneの4byteだけ
			tls := unpackTLSHandshake(v[2:6])
			proto := TLSProtocol{
				RHeader:           rHeader,
				HandshakeProtocol: tls,
			}
			protocolsByte = append(protocolsByte, v[2:6]...)
			protocols = append(protocols, proto)
		}
	}
	return protocols, protocolsByte
}

func starFromClientHello(sendfd int, sendInfo TCPIP) error {
	clienthelloPacket := NewTCPIP(sendInfo)
	destIp := iptobyte(sendInfo.DestIP)

	// Client Helloを送る
	addr := setSockAddrInet4(destIp, int(sendInfo.DestPort))
	SendIPv4Socket(sendfd, clienthelloPacket, addr)
	fmt.Printf("Send TLS Client Hello to : %s\n", sendInfo.DestIP)

	var recvtcp TCPHeader
	var ack TCPIP
	var tlsProto []TLSProtocol
	var tlsBytes []byte
	var tcpBytes []byte

	_ = tlsBytes

	var handshake_messages []byte
	handshake_messages = append(handshake_messages, sendInfo.Data[5:]...)

	for {
		recvBuf := make([]byte, 65535)
		_, _, err := syscall.Recvfrom(sendfd, recvBuf, 0)
		if err != nil {
			log.Fatalf("read err : %v", err)
		}
		// IPヘッダをUnpackする
		ip := parseIP(recvBuf[0:20])
		if bytes.Equal(ip.Protocol, []byte{0x06}) && bytes.Equal(ip.SourceIPAddr, destIp) {
			// IPヘッダを省いて20byte目からのTCPパケットをパースする
			recvtcp = parseTCP(recvBuf[20:])

			if recvtcp.ControlFlags[0] == ACK && bytes.Equal(recvtcp.SourcePort, uintTo2byte(sendInfo.DestPort)) {
				fmt.Printf("Recv ACK from %s\n", sendInfo.DestIP)
				time.Sleep(10 * time.Millisecond)
			} else if recvtcp.ControlFlags[0] == PSHACK && bytes.Equal(recvtcp.SourcePort, uintTo2byte(sendInfo.DestPort)) {
				fmt.Printf("Recv PSHACK from %s\n", sendInfo.DestIP)
				//fmt.Printf("TCP Data : %s\n", printByteArr(recvtcp.TCPData))

				tcpLength := uint32(sumByteArr(ip.TotalPacketLength)) - 20
				tcpLength = tcpLength - uint32(recvtcp.HeaderLength[0]>>4<<2)

				tcpBytes = append(tcpBytes, recvtcp.TCPData[0:tcpLength]...)
				//fmt.Printf("PSHACK TCP Data : %s\n", printByteArr(tlsbyte))

				tlsProto, tlsBytes = unpackTLSPacket(tcpBytes)
				//pp.Println(tlsProto)

				time.Sleep(10 * time.Millisecond)

				ack = TCPIP{
					DestIP:    sendInfo.DestIP,
					DestPort:  sendInfo.DestPort,
					TcpFlag:   "ACK",
					SeqNumber: recvtcp.AcknowlegeNumber,
					AckNumber: calcSequenceNumber(recvtcp.SequenceNumber, tcpLength),
				}
				ackPacket := NewTCPIP(ack)
				// ServerHelloを受信したことに対してACKを送る
				SendIPv4Socket(sendfd, ackPacket, addr)

				for _, v := range tlsProto {
					switch v.HandshakeProtocol.(type) {
					case ServerHelloDone:
						fmt.Printf("Recv PSHACK ServerHello,Certificate,ServerHelloDone from %s\n", sendInfo.DestIP)
						//break
					}
				}
				break
			}
		}
	}

	//handshake_messages = append(handshake_messages, tlsBytes...)
	//_, err := sendClientKeyExchangeToFinish(sendfd, TCPandServerHello{
	//	ACKFromClient:      ack,
	//	TLSProcotocol:      tlsProto,
	//	TLSProcotocolBytes: handshake_messages,
	//})
	//if err != nil {
	//	log.Fatal(err)
	//}
	return nil
}

/*func sendClientKeyExchangeToFinish(sendfd int, serverhello TCPandServerHello) ([]byte, error) {
	var serverRandom []byte
	var pubkey *rsa.PublicKey

	for _, v := range serverhello.TLSProcotocol {
		switch proto := v.HandshakeProtocol.(type) {
		case ServerHello:
			serverRandom = proto.Random
		case ServerCertificate:
			_, ok := proto.Certificates[0].PublicKey.(*rsa.PublicKey)
			if !ok {
				log.Fatalf("cast pubkey err : %v\n", ok)
			}
			pubkey = proto.Certificates[0].PublicKey.(*rsa.PublicKey)
		}
	}

	var clientKeyExchange ClientKeyExchange
	clientKeyExchangeBytes, premasterBytes := clientKeyExchange.NewClientKeyExchange(pubkey)
	serverhello.TLSProcotocolBytes = append(serverhello.TLSProcotocolBytes, clientKeyExchangeBytes[5:]...)

	changeCipher := NewChangeCipherSpec()

	masterBytes := MasterSecretInfo{
		PreMasterSecret: premasterBytes,
		ServerRandom:    serverRandom,
		ClientRandom:    serverhello.ClientHelloRandom,
	}

	verifyData, keyblock, _ := createVerifyData(masterBytes, CLientFinishedLabel, serverhello.TLSProcotocolBytes)
	// finished messageを作成する、先頭にヘッダを入れてからverify_dataを入れる
	// 作成された16byteがplaintextとなり暗号化する
	finMessage := []byte{HandshakeTypeFinished}
	finMessage = append(finMessage, uintTo3byte(uint32(len(verifyData)))...)
	finMessage = append(finMessage, verifyData...)
	fmt.Printf("finMessage : %x\n", finMessage)

	rheader := NewTLSRecordHeader("Handshake", uint16(len(finMessage)))
	//encryptFin := encryptMessage(rheader, keyblock.ClientWriteIV, finMessage, keyblock.ClientWriteKey)

	var all []byte
	all = append(all, clientKeyExchangeBytes...)
	all = append(all, changeCipher...)
	//all = append(all, encryptFin...)

	fin := TCPIP{
		DestIP:    LOCALIP,
		DestPort:  LOCALPORT,
		TcpFlag:   "PSHACK",
		SeqNumber: serverhello.ACKFromClient.SeqNumber,
		AckNumber: serverhello.ACKFromClient.AckNumber,
		Data:      all,
	}

	finished := NewTCPIP(fin)

	destIp := iptobyte(fin.DestIP)

	// Finished message を送る
	addr := setSockAddrInet4(destIp, LOCALPORT)
	err := SendIPv4Socket(sendfd, finished, addr)
	if err != nil {
		return nil, fmt.Errorf("send PSHACK packet err : %v", err)
	}
	fmt.Printf("Send Finished message to : %s\n", LOCALIP)

	return all, nil
}
*/

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

// Proxy Protocol v2 starts with a fixed 16-byte “header”:
//
// +-------- 12 bytes  -------+-----1-----+-----1-----+-----2------+
// | Signature (PP2_SIG)      | Ver+Cmd   | Fam+Proto | Len (BE)   |
// | = 0D0A0D0A 00 0D0A51 52 4F 54 0A                              |
// +---------------------------------------------------------------+
//
// 	•	Signature (12 bytes): 0x0D 0A 0D 0A 00 0D 0A 51 52 4F 54 0A
//	•	Version + Command (1 byte): Highest 4 bits = 0x2 for version 2; next 4 bits for command (e.g., 0x1 = PROXY).
//	•	Family + Protocol (1 byte): For our example, we expect:
//	•	0x11 = AF_INET (IPv4) + STREAM (TCP)
//	•	0x21 = AF_INET6 (IPv6) + STREAM (TCP)
//	•	Len (2 bytes, big-endian): The length of the “address information” that follows immediately in the stream.
//
// After the initial 16 bytes, you have the address block, whose size is Len:
//	•	For IPv4 (AF_INET): 12 bytes total
//	1.	4 bytes: Source IP
//	2.	4 bytes: Destination IP
//	3.	2 bytes: Source port
//	4.	2 bytes: Destination port
//	•	For IPv6 (AF_INET6): 36 bytes total
//	1.	16 bytes: Source IP
//	2.	16 bytes: Destination IP
//	3.	2 bytes: Source port
//	4.	2 bytes: Destination port

// ppv2Signature is the 12-byte signature for Proxy Protocol v2
var ppv2Signature = [12]byte{0x0D, 0x0A, 0x0D, 0x0A, 0x00, 0x0D, 0x0A, 0x51, 0x52, 0x4F, 0x54, 0x0A}

// checkSignature compares the first 12 bytes with the known PPv2 signature.
func checkSignature(sig []byte) bool {
	if len(sig) != len(ppv2Signature) {
		return false
	}
	for i := 0; i < len(ppv2Signature); i++ {
		if sig[i] != ppv2Signature[i] {
			return false
		}
	}
	return true
}

// handle is called for each new TCP connection.
func handle(raw net.Conn) {
	defer raw.Close()

	// Set a short read deadline to avoid hanging if no data arrives
	_ = raw.SetReadDeadline(time.Now().Add(5 * time.Second))

	// 1. Read the first 16 bytes for the PPv2 header
	header := make([]byte, 16)
	if _, e := io.ReadFull(raw, header); e != nil {
		log.Printf("Error reading initial header: %v\n", e)
		return
	}

	// 2. Check the 12-byte signature
	if !checkSignature(header[:12]) {
		// If signature doesn't match, we might have plain TCP (no proxy protocol).
		// In a real server, you might either reject or treat this as non-proxied.
		log.Println("No valid Proxy Protocol v2 signature detected; treat as non-proxied?")
		// We can either handle the data as normal or close:
		// Here, we just log and return.
		return
	}

	// 3. Parse version/command byte (header[12]) and family/protocol byte (header[13])
	verCmd := header[12]
	famProto := header[13]

	// The top 4 bits of verCmd must be 0x2 for PPv2
	// Low 4 bits is the command (0x1 = PROXY, 0x0 = LOCAL, etc.)
	version := verCmd >> 4
	if version != 2 {
		log.Printf("Unsupported Proxy Protocol version: %d\n", version)
		return
	}

	cmd := verCmd & 0x0F
	if cmd != 0x1 {
		// 0x1 = PROXY (the only one that carries client info)
		// 0x0 = LOCAL, meaning no address data, etc.
		log.Printf("Command is %d, not PROXY; ignoring.\n", cmd)
		return
	}

	// 4. Extract address length
	addrLen := binary.BigEndian.Uint16(header[14:16])

	// 5. Read the address block
	addrBlock := make([]byte, addrLen)
	if _, err := io.ReadFull(raw, addrBlock); err != nil {
		log.Printf("Error reading address block: %v\n", err)
		return
	}

	// 6. Parse IP addresses based on famProto
	//    famProto high nibble = 0x1 => AF_INET (IPv4), 0x2 => AF_INET6 (IPv6)
	//    low nibble often = 0x1 => STREAM (TCP)
	family := famProto >> 4
	var srcIP, dstIP net.IP
	var srcPort, dstPort uint16

	switch family {
	case 0x1: // IPv4
		if len(addrBlock) < 12 {
			log.Println("Not enough data for IPv4 address block.")
			return
		}
		srcIP = net.IPv4(addrBlock[0], addrBlock[1], addrBlock[2], addrBlock[3])
		dstIP = net.IPv4(addrBlock[4], addrBlock[5], addrBlock[6], addrBlock[7])
		srcPort = binary.BigEndian.Uint16(addrBlock[8:10])
		dstPort = binary.BigEndian.Uint16(addrBlock[10:12])

	case 0x2: // IPv6
		if len(addrBlock) < 36 {
			log.Println("Not enough data for IPv6 address block.")
			return
		}
		srcIP = net.IP(addrBlock[0:16])
		dstIP = net.IP(addrBlock[16:32])
		srcPort = binary.BigEndian.Uint16(addrBlock[32:34])
		dstPort = binary.BigEndian.Uint16(addrBlock[34:36])

	default:
		log.Printf("Unrecognized address family: 0x%X\n", family)
		return
	}

	// We have the client’s real IP/port (srcIP:srcPort)
	clientAddr := fmt.Sprintf("%s:%d", srcIP.String(), srcPort)
	destAddr := fmt.Sprintf("%s:%d", dstIP.String(), dstPort)
	log.Printf("Got Proxy Protocol v2 connection from %s -> %s\n", clientAddr, raw.LocalAddr().String())
	log.Printf("Destination %s -> %s\n", destAddr, raw.LocalAddr().String())

	// 7. The rest of the connection is the actual application data
	//    We'll show a quick example: reading some data from the client.
	buf := make([]byte, 1024)
	n, err := raw.Read(buf)
	if err != nil && err != io.EOF {
		log.Printf("Read error: %v\n", err)
		return
	}

	log.Printf("Received %d bytes from %s\n", n, clientAddr)

	// Echo back or handle application logic as needed
	raw.Write([]byte("Hello from server!\n"))
}

func main() {
	// Listen on TCP :9000
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Error listening on :9000: %v", err)
	}
	defer ln.Close()
	log.Println("Server listening on :9000...")

	// Accept loop
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Accept error: %v\n", err)
			continue
		}
		go handle(conn)
	}
}

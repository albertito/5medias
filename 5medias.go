// 5medias is a simple SOCKS5 proxy.
//
// https://www.ietf.org/rfc/rfc1928.txt
//
// It only supports TCP CONNECT commands, and static username/password
// authentication.
//
// This authentication is only useful to prevent the most basic unauthorized
// access, but it is not really secure and it's extremely easy to sniff. Don't
// rely on it for real security.
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"time"

	"blitiri.com.ar/go/log"
)

// Command-line flags.
var (
	addr     = flag.String("addr", ":1080", "address to listen on")
	username = flag.String("username", "", "username to expect")
	password = flag.String("password", "", "password to expect")
)

func main() {
	flag.Parse()
	log.Init()

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("error listening: %v", err)
	}

	log.Infof("listening on %s", *addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Errorf("error accepting: %v", err)
			continue
		}

		c := &Conn{conn: conn}
		go c.Handle()
	}
}

type Conn struct {
	conn net.Conn
}

func (c *Conn) Logf(f string, a ...interface{}) {
	log.Log(log.Info, 1, "%v: %s",
		c.conn.RemoteAddr(), fmt.Sprintf(f, a...))
}

func (c *Conn) Handle() {
	defer c.conn.Close()

	c.Logf("connected")
	if err := c.handshake(); err != nil {
		c.Logf("handshake error: %v", err)
		return
	}

	dstAddr, err := c.getRequest()
	if err != nil {
		c.Logf("request error: %v", err)
		return
	}

	c.Logf("dial %q", dstAddr)
	dstConn, err := net.DialTimeout("tcp", dstAddr, 10*time.Second)
	if err != nil {
		c.Logf("outgoing connection error: %v", err)
		c.reply(5) // Connection refused.
		return
	}
	defer dstConn.Close()

	if dstConn.LocalAddr().(*net.TCPAddr).IP.IsLoopback() {
		c.Logf("loopback connection denied")
		c.reply(2) // Connection not allowed by ruleset.
		return
	}

	c.reply(0) // Success.

	c.Logf("proxying begin")
	bidirCopy(c.conn, dstConn)
	c.Logf("proxying end")
}

func (c *Conn) handshake() error {
	hdr := struct {
		Version  byte
		NMethods byte
	}{}
	if err := binary.Read(c.conn, binary.BigEndian, &hdr); err != nil {
		return err
	}

	if hdr.Version != 5 {
		return fmt.Errorf("invalid version")
	}

	methods := make([]byte, hdr.NMethods)
	if err := binary.Read(c.conn, binary.BigEndian, methods); err != nil {
		return err
	}

	if *username == "" {
		// Reply saying that we are ok with no authentication (#0).
		c.conn.Write([]byte{5, 0})
	} else {
		// Check if username/password (method 2) was offered.
		if !bytes.Contains(methods, []byte{2}) {
			c.conn.Write([]byte{5, 0xff})
			return fmt.Errorf("username/password method not offered")
		}

		// Reply saying that we accept user/password method (#2).
		c.conn.Write([]byte{5, 2})

		if err := c.auth(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Conn) auth() error {
	ver, err := c.readByte()
	if err != nil {
		return err
	}
	if ver != 1 {
		return fmt.Errorf("unknown username/password version")
	}

	cUser, err := c.readBuf()
	if err != nil {
		return err
	}

	cPassword, err := c.readBuf()
	if err != nil {
		return err
	}

	if string(cUser) != *username || string(cPassword) != *password {
		// Introduce a small delay to mitigate blatant brute force attempts.
		time.Sleep(200 * time.Millisecond)
		c.conn.Write([]byte{1, 0xff})
		return fmt.Errorf("invalid username/password")
	}

	c.conn.Write([]byte{1, 0})

	return nil
}

func (c *Conn) getRequest() (string, error) {
	hdr := struct {
		Version  byte
		Command  byte
		Reserved byte
		AddrType byte
	}{}
	if err := binary.Read(c.conn, binary.BigEndian, &hdr); err != nil {
		return "", err
	}

	if hdr.Version != 5 {
		return "", fmt.Errorf("invalid version")
	}
	if hdr.Command != 1 {
		return "", fmt.Errorf("unsupported command %v", hdr.Command)
	}

	addrLen := 0
	switch hdr.AddrType {
	case 1: // ipv4
		addrLen = net.IPv4len
	case 3: // domain name
		addrLen = 0
	case 4: // ipv6
		addrLen = net.IPv6len
	default:
		return "", fmt.Errorf("unsupported address type %v", hdr.AddrType)
	}

	addr := ""
	if addrLen == 0 {
		domain, err := c.readBuf()
		if err != nil {
			return "", err
		}
		addr = string(domain)
	} else {
		raw := make([]byte, addrLen)
		if _, err := c.conn.Read(raw); err != nil {
			return "", err
		}
		ip := net.IP(raw)

		addr = ip.String()
	}

	port := uint16(0)
	if err := binary.Read(c.conn, binary.BigEndian, &port); err != nil {
		return "", err
	}

	return net.JoinHostPort(addr, fmt.Sprintf("%v", port)), nil
}

func (c *Conn) reply(result byte) {
	c.conn.Write([]byte{
		5, // version
		result,
		0,          // reserved
		1,          // addr type (ipv4, for convenience)
		0, 0, 0, 0, // ipv4 (0.0.0.0)
		0, 0, // port 0 (2 bytes)
	})
}

func (c *Conn) readByte() (byte, error) {
	b := make([]byte, 1)
	_, err := c.conn.Read(b)
	return b[0], err
}

func (c *Conn) readBuf() ([]byte, error) {
	bufLen, err := c.readByte()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, bufLen)
	_, err = c.conn.Read(buf)
	return buf, err
}

func bidirCopy(src, dst io.ReadWriter) {
	done := make(chan bool, 2)

	go func() {
		io.Copy(src, dst)
		done <- true
	}()

	go func() {
		io.Copy(dst, src)
		done <- true
	}()

	// Return when one of the two completes.
	// The other goroutine will remain alive, it is up to the caller to create
	// the conditions to complete it (e.g. by closing one of the sides).
	<-done
}

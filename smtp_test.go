package smtp

import (
	//"bytes"
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
)

const (
	hostport = "127.0.0.1:10025"
)

var message = []byte("From: sender@example.com\r\nSubject: test\r\n\r\nHello\r\n")

// testConn is used to implement test smtp server
type testConn struct {
	//b *bytes.Buffer
	b io.Writer
	r *bufio.Reader
	w io.Writer
	t *testing.T
}

func newTestConn(t *testing.T, c net.Conn) *testConn {
	return &testConn{
		//b: &bytes.Buffer{},
		b: os.Stderr,
		r: bufio.NewReader(c),
		w: c,
		t: t,
	}
}

func (c *testConn) reply(s string) {
	fmt.Fprint(c.b, "<- ", s, "\r\n")
	fmt.Fprint(c.w, s, "\r\n")
}

func (c *testConn) expect(cmd string) {
	line, err := c.r.ReadString('\n')
	if err != nil {
		c.t.Fatalf("%v", err)
	}
	fmt.Fprint(c.b, "-> ", line)
	if line[len(line)-2] == '\r' {
		line = line[0 : len(line)-2]
	} else {
		line = line[0 : len(line)-1]
	}
	if strings.HasPrefix(line, cmd) == false {
		//fmt.Printf(c.b.String())
		c.t.Errorf("Expected %s but got %s", cmd, line[0:len(cmd)])
	}
}

func (c *testConn) readData() {
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			c.t.Fatalf("%v", err)
		}
		if line[0] == '.' {
			fmt.Fprint(c.b, "-> ", line)
			if line[1] == '\r' {
				if line[2] == '\n' {
					break
				} else {
					c.t.Fatalf("Expected \\n after \".\\r\"")
				}
			} else {
				if line[2] != '.' {
					c.t.Fatalf("Expected \\r or '.' after '.'")
				}
			}
		}
	}
}

// server with responses and expected commands specified in f
func server(t *testing.T, f func(*testConn)) {

	listener, err := net.Listen("tcp", hostport)
	if err != nil {
		t.Fatalf("%v", err)
	}
	//defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Fatalf("%v", err)
		}
		//defer conn.Close()

		f(newTestConn(t, conn))

		conn.Close()
		listener.Close()
	}()
}

func TestSendMailNoPipelining(t *testing.T) {
	server(t, func(c *testConn) {
		c.reply("220 localhost ESMTP")
		c.expect("EHLO")
		c.reply("250 OK")
		c.expect("MAIL")
		c.reply("250 OK")
		c.expect("RCPT")
		c.reply("250 OK")
		c.expect("DATA")
		c.reply("354 End data with <CR><LF>.<CR><LF>")
		c.readData()
		c.reply("250 OK")
		c.expect("QUIT")
		c.reply("221 localhost closing connection")
	})

	err := SendMail(hostport, nil, "mike@sender.com", []string{"john@receiver.com"}, message)
	if err != nil {
		t.Fatalf("%T: %v", err, err)
	}
}

func TestSendMailWithPipelining(t *testing.T) {
	server(t, func(c *testConn) {
		c.reply("220 localhost ESMTP")
		c.expect("EHLO")
		c.reply("250-localhost")
		c.reply("250 PIPELINING")
		c.expect("MAIL")
		c.expect("RCPT")
		c.expect("DATA")
		c.reply("250 OK")
		c.reply("250 OK")
		c.reply("354 End data with <CR><LF>.<CR><LF>")
		c.readData()
		c.reply("250 OK")
		c.expect("QUIT")
		c.reply("221 localhost closing connection")
	})

	err := SendMail(hostport, nil, "mike@sender.com", []string{"john@receiver.com"}, message)
	if err != nil {
		t.Fatalf("%T: %v", err, err)
	}
}

func TestSendMailWithRecipientErrors(t *testing.T) {
	server(t, func(c *testConn) {
		c.reply("220 localhost ESMTP")
		c.expect("EHLO")
		c.reply("250-localhost")
		c.reply("250 PIPELINING")
		c.expect("MAIL")
		c.expect("RCPT")
		c.expect("RCPT")
		c.expect("RCPT")
		c.expect("DATA")
		c.reply("250 OK")
		c.reply("250 OK")
		c.reply("452 Over quota")
		c.reply("550 User unknown")
		c.reply("354 End data with <CR><LF>.<CR><LF>")
		c.readData()
		c.reply("250 OK")
		c.expect("QUIT")
		c.reply("221 localhost closing connection")
	})

	err := SendMail(hostport, nil, "mike@sender.com",
		[]string{"john@receiver.com", "fred@receiver.com", "sara@receiver.com"}, message)
	me, ok := err.(MultiError)
	if ok == false {
		t.Fatalf("%T: %v", err, err)
	}
	if me[0] != nil {
		t.Fatalf("%T: %v", me[0], me[0])
	}
	if me[1] == nil || me[2] == nil {
		t.Fatalf("Expected error after RCPT")
	}
	if strings.Contains(me[1].Error(), "452") == false {
		t.Fatalf("%T: %v", me[1], me[1])
	}
	if strings.Contains(me[2].Error(), "550") == false {
		t.Fatalf("%T: %v", me[2], me[2])
	}
}

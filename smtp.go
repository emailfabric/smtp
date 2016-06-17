/*
Package smtp is a replacement for net/smtp with improvements and additions.
*/
package smtp

import (
	"net"
	"net/smtp"
	"net/textproto"
	"os"
	"strings"
	"time"
)

var (
	// ConnectTimeout is shortened in some mtas, the OS may also impose shorter timeouts
	ConnectTimeout = 5 * time.Minute
	GreetingTimeout = 5 * time.Minute
)

// Client embeds a smtp.Client and provides additional member functions
type Client struct {
	*smtp.Client
}

// Dial connects to addr from the default IP, waits for the banner greeting and returns a new Client.
func Dial(addr string) (*Client, error) {
	return DialFrom(addr, nil)
}

// DialFrom connects to addr from the specified localIP, waits for the banner greeting and returns a new Client.
func DialFrom(addr string, localIP net.IP) (*Client, error) {

	var serverName string
	if strings.LastIndexByte(addr, ':') == -1 {
		serverName = addr
		addr += ":25"
	} else {
		serverName, _, _ = net.SplitHostPort(addr)
	}

	dialer := &net.Dialer{Timeout: ConnectTimeout}
	if localIP != nil {
		dialer.LocalAddr = &net.TCPAddr{IP: localIP}
	}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return NewClient(conn, serverName)
}

// NewClient waits for the 220 banner greeting and returns a new Client using 
// an existing connection and host as a server name to be used when authenticating.
func NewClient(conn net.Conn, serverName string) (*Client, error) {

	conn.SetReadDeadline(time.Now().Add(GreetingTimeout))
	c, err := smtp.NewClient(conn, serverName)
	if err != nil {
		return nil, err
	}
	conn.SetReadDeadline(time.Time{})  // remove deadline
	return &Client{c}, nil
}

// SendMail connects to the server at addr, switches to TLS if possible, 
// authenticates with the optional mechanism a if possible, and then sends 
// an email from address from, to addresses to, with message msg. 
// The addr must include a port, as in "mail.example.com:smtp".
func SendMail(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {

	c, err := Dial(addr)
	if err != nil {
		return err
	}
	defer c.Close() // c.Quit()?

	hostname, _ := os.Hostname()
	err = c.Session(hostname, "", auth)
	if err != nil {
		return err
	}

	wc, tranErr := c.Transaction(from, to)
	if wc == nil && tranErr != nil {
		return tranErr
	}

	_, err = wc.Write(msg)
	if err != nil {
		return MergeError(tranErr, err)
	}
	err = wc.Close()
	if err != nil {
		return MergeError(tranErr, err)
	}

	c.Quit()
	return tranErr
}

// IsPermanent returns true if err is a reply with 5XX status code
func IsPermanent(err error) bool {
    if tpe, ok := err.(*textproto.Error); ok {
        return tpe.Code / 100 == 5
    }
    return false
}

// PlainAuth returns an Auth that implements the PLAIN authentication
// mechanism as defined in RFC 4616.
// The returned Auth uses the given username and password to authenticate
// on TLS connections to host and act as identity. Usually identity will be
// left blank to act as username.
func PlainAuth(identity, username, password, host string) smtp.Auth {
	return smtp.PlainAuth(identity, username, password, host)
}

// CRAMMD5Auth returns an Auth that implements the CRAM-MD5 authentication
// mechanism as defined in RFC 2195.
// The returned Auth uses the given username and secret to authenticate
// to the server using the challenge-response mechanism.
func CRAMMD5Auth(username, secret string) smtp.Auth {
	return smtp.CRAMMD5Auth(username, secret)
}

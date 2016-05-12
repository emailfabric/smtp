package smtp

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	
	"github.com/pkg/errors"	
)

// Session initiates an SMTP session.
func (c *Client) Session(localName string, serverName string, auth smtp.Auth) error {

	// note that smtp.Client#hello fallbacks to HELO after any error
	// instead it should fallback only after 500 (or 502)

	err := c.Hello(localName)
	if err != nil {
		return errors.Wrap(err, "EHLO command failed")
	}

	if ok, _ := c.Extension("STARTTLS"); ok {
		// must set either ServerName or InsecureSkipVerify
		config := tls.Config{}
		if serverName != "" {
			config.ServerName = serverName
		} else {
			config.InsecureSkipVerify = true
		}
		err := c.StartTLS(&config)
		if err != nil {
			// If the client receives the 454 response, the client must decide
			// whether or not to continue the SMTP session.  Such a decision is
			// based on local policy.
			return errors.Wrap(err, "tls connection failed")
		}
		// handshake is done at first I/O, do it now?
	}

	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			err := c.Auth(auth)
			if err != nil {
				return errors.Wrap(err, "authentication failed")
			}
		} else {
			return fmt.Errorf("authentication requested but not supported")
		}
	}
	return nil
}

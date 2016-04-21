# smtp

[![](https://godoc.org/github.com/emailfabric/smtp?status.svg)](http://godoc.org/github.com/emailfabric/smtp)

Go library that improves and extends net/smtp.

The SMTP client in net/smtp already supports a number of features:

* Support for [PLAIN](https://tools.ietf.org/html/rfc4616) authentication.
* Support for [CRAM-MD5](https://tools.ietf.org/html/rfc2195) authentication.
* Support for [STARTTLS](https://www.ietf.org/rfc/rfc3207) encryption.

But it also misses some features which limit it's application to simple forwarding scenarios. 

This library adds the following features:

* Support binding to specific local IP.
* Use port 25 if no port specified instead of always requiring port.
* Use os.Hostname as default EHLO name instead of using "localhost" as default.
* Disable TLS domain verification, unless serverName is specified.
* Provide Session() and Transaction() members as intermediate level API.
* Support for [PIPELINING](https://tools.ietf.org/html/rfc2920) extension.
* Send DATA if at least one RCPT is accepted instead of returning after the first failed recipient.

TODOS:

* Remove net/smtp dependency for smtp.Auth. 
* Do not send HELO after EHLO with 4XX error.
* Command-response timeouts according to RFC.
* More context information in errors.
* Issue RSET before next transaction if last transaction failed.
* Less picky about response codes, the first digit is conclusive.

## Usage

### Forward one message

To connect to an MX, forward a message, and disconnect:

	err = smtp.SendMail(mx, sender, recipients, message, nil)
	
### Bulk email delivery

To connect to an MX from a specific local IP address:

	client, err := smtp.DialFrom("gmail-smtp-in.l.google.com", net.ParseIP("1.2.3.4"))

To initiate a session, using "opportunistic" TLS and no authentication:

	err := client.Session("mail.example.com", "", nil)
	
To start a transaction, using pipelining if available:

	wc, err := client.Transaction(sender, recipients)

If multiple recipients are specified, some can be rejected and some can be accepted. If the transaction succeeds and at least one recipient was accepted a non-nil io.WriteCloser is returned. A non-nil err indicates that the transaction failed, or that at least one recipient was rejected.

To send the message from an io.Reader:

	_, err := io.Copy(wc, reader)
	if err != nil {
		return err
	}
    err = wc.Close()

Then another transaction can be started or the session can be terminated:

	client.Quit()
	
    



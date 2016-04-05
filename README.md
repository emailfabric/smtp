# smtp

Go library that improves and extends net/smtp.

The SMTP client in net/smtp already supports a number of features:

* Support for [PLAIN](https://tools.ietf.org/html/rfc4616) authentication.
* Support for [CRAM-MD5](https://tools.ietf.org/html/rfc2195) authentication.
* Support for STARTTLS.

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

## Usage

### Forward one message

To connect to an MX, forward a message, and disconnect:

	err = smtp.SendMail(mx, sender, recipients, message, nil)
	
### Bulk email delivery

To connect to an MX from a specific local IP address:

	client, err := smtp.DialFrom("gmail-smtp-in.l.google.com", net.ParseIP("1.2.3.4"))

To initiate a session, using "opportunistic" TLS and no authentication:

	err = client.Session("mail.example.com", "", nil)
	
To start a transaction:

	wc, tranErr := client.Transaction(sender, recipients)

The returned error can indicate that some recipients failed and others were accepted.
	
To send the message:

	_, err = wc.Write(msg)
	
To send the end-of-message sequence: 

    err = wc.Close()

Then another transaction can be started or the session can be terminated:

	client.Quit()
	
    



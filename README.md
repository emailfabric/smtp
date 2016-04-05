# smtp

Library that improves and extends net/smtp.

* Support binding to specific local IP.
* Use port 25 if no port specified instead of always requiring port.
* Use os.Hostname as default EHLO name instead of using "localhost" as default.
* Disable TLS domain verification, unless serverName is specified.
* Support for PIPELINING extension.
* Send DATA if at least one RCPT is accepted instead of returning after the first failed recipient.

TODO: do not send HELO after 4XX error after EHLO

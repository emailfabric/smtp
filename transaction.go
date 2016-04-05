package smtp

import (
    "fmt"
    "io"
    "net/textproto"
)

// MultiError is returned by batch operations when there are errors with
// particular elements. Errors will be in a one-to-one correspondence with
// the input elements; successful elements will have a nil entry.
type MultiError []error

func (m MultiError) Error() string {
	s, n := "", 0
	for _, e := range m {
		if e != nil {
			if n == 0 {
				s = e.Error()
			}
			n++
		}
	}
	switch n {
	case 0:
		return "(0 errors)"
	case 1:
		return s
	case 2:
		return s + " (and 1 other error)"
	default:
    	return fmt.Sprintf("%s (and %d other errors)", s, n-1)
	}
}

// merge sets nil items to err and returns either m (if m has non nil) or err (if m has all nils)
func (m MultiError) merge(err error) error {
    isMulti := false
    for i := 0; i < len(m); i++ {
        if m[i] == nil {
            m[i] = err
        } else {
            isMulti = true
        }
    }
    if isMulti {
        return m
    } else {
        return err
    }
} 

func MergeError(prevErr error, newErr error) error {
    if me, ok := prevErr.(MultiError); ok {
        return me.merge(newErr)
    } else {
        // return previous error if not nil
        if prevErr == nil {
            return newErr
        } else {
            return prevErr
        }
    }
}

// Transaction starts a new transaction. 
// A transaction can "partially fail" if at least one, but not all recipients failed.
// If MAIL and at least one RCPT succeeded, the DATA command is sent and io.WriteCloser is returned if DATA succeeded.
// If at least one recipient has failed, an error is returned. If more than one recipient failed a MultiError is returned.
func (c *Client) Transaction(from string, to []string) (io.WriteCloser, error) {

	if ok, _ := c.Extension("PIPELINING"); ok {
	    return c.pipelining(from, to)
    } 
    
    // fallback to normal lockstep transaction
    
    err := c.Mail(from)
    if err != nil {
        return nil, err
    }

    rcptErr := make(MultiError, len(to))
    var failed int
    for i, addr := range to {
        rcptErr[i] = c.Rcpt(addr)
        if rcptErr[i] != nil {
    	    failed++
    	}
    }

    // all recipients failed?
    if failed == len(to) {
	    if len(rcptErr) == 1 {
	        return nil, rcptErr[0]
	    } else {
	        return nil, rcptErr
	    }
	}

    return c.Data()
}

type dataCloser struct {
	r *textproto.Reader
	io.WriteCloser
}

func (d *dataCloser) Close() error {
	d.WriteCloser.Close()
	_, _, err := d.r.ReadResponse(250)
	return err
}

// pipelining starts a transaction with pipelining
func (c *Client) pipelining(from string, to []string) (io.WriteCloser, error) {

    //
    // step 1: send commands in one stroke
    //

    cmdStr := "MAIL FROM:<%s>\r\n"
	if ok, _ := c.Extension("8BITMIME"); ok {
		cmdStr += " BODY=8BITMIME"
	}

    // textproto.Conn#Cmd is avoided because it expects that textproto.Pipeline is used
    // textproto.Conn#PrintfLine is avoided for MAIL and RCPT because it does unneeded flush
    // note that SMTP pipelining is different from textproto.Pipeline
    // the first is to send requests as a group, the latter is for concurrent requests
    
	_, err := fmt.Fprintf(c.Text.Writer.W, cmdStr, from)
	if err != nil {
		return nil, err
	}

    for _, addr := range to {
    	_, err := fmt.Fprintf(c.Text.Writer.W, "RCPT TO:<%s>\r\n", addr)
        if err != nil {
            return nil, err
        }
    }
    
    err = c.Text.PrintfLine("DATA")  // CRLF added and flushed
    if err != nil {
        return nil, err
    }
    
    //
    // step 2: collect replies from all commands
    //

	_, _, mailErr := c.Text.ReadResponse(250)

    rcptErr := make(MultiError, len(to))
    var failed int
	for i := 0; i < len(to); i++ {
	    _, _, rcptErr[i] = c.Text.ReadResponse(25)
    	if rcptErr[i] != nil {
    	    failed++
    	}
	}

	_, _, dataErr := c.Text.ReadResponse(354)
	
	//
	// step 3: check replies
	//
	
	if mailErr != nil {
	    return nil, mailErr
	}

	if failed == len(to) {
	    // Special case if the server rejected all recipients but accepted the DATA 
        // command. Client should just send "." as per RFC 2197.
        if dataErr == nil {
            c.Text.DotWriter().Close()
        }
	    if len(rcptErr) == 1 {
	        return nil, rcptErr[0]
	    } else {
	        return nil, rcptErr // possibly different replies
	    }
	}
	
	if dataErr != nil {
	    // merge with any failed recipients
	    return nil, rcptErr.merge(dataErr)
	}

	return &dataCloser{&c.Text.Reader, c.Text.DotWriter()}, rcptErr.merge(nil)
}

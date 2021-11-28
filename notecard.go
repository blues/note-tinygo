// Copyright 2017 Blues Inc.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

package tinynote

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// DefaultI2CAddress is Our default I2C address
const DefaultI2CAddress = 0x17

// ErrCardIo is the card I/O error suffix
const ErrCardIo = "{io}"

// ErrTimeout is the card timeout error suffix
const ErrTimeout = "{timeout}"

// InitialDebugMode is the debug mode that the context is initialized with
var InitialDebugMode = false

// Protect against multiple concurrent callers
var transLock sync.RWMutex

// SerialTimeoutMs is the response timeout for Notecard serial communications.
var SerialTimeoutMs = 10000

// CardI2CMax controls chunk size that's socially appropriate on the I2C bus.
// It must be 1-253 bytes as per spec (which allows space for the 2-byte header in a 255-byte read)
const CardI2CMax = 253

// I2CTxFn is the function to write and read from the I2C Port
type I2CTxFn func(i2cAddress uint16, writebuf []byte, readbuf []byte) (err error)

// UARTReadFn is the function to read from the UART port
type UARTReadFn func(data []byte) (n int, err error)

// UARTWriteFn is the function to write to the UART port
type UARTWriteFn func(data []byte) (n int, err error)

// The notecard is a real-time device that has a fixed size interrupt buffer.  We can push data
// at it far, far faster than it can process it, therefore we push it in segments with a pause
// between each segment.

// CardRequestSerialSegmentMaxLen (golint)
const CardRequestSerialSegmentMaxLen = 250

// CardRequestSerialSegmentDelayMs (golint)
const CardRequestSerialSegmentDelayMs = 250

// CardRequestI2CSegmentMaxLen (golint)
const CardRequestI2CSegmentMaxLen = 250

// CardRequestI2CSegmentDelayMs (golint)
const CardRequestI2CSegmentDelayMs = 250

// RequestSegmentMaxLen (golint)
var RequestSegmentMaxLen = -1

// RequestSegmentDelayMs (golint)
var RequestSegmentDelayMs = -1

// Context for the port that is open
type Context struct {

	// True to emit trace output
	Debug bool

	// Disable generation of User Agent object
	DisableUA bool

	// Class functions
	CloseFn       func(context *Context)
	ResetFn       func(context *Context) (err error)
	TransactionFn func(context *Context, noResponse bool, reqJSON []byte) (rspJSON []byte, err error)

	// I/O functions
	i2cTxFn     I2CTxFn
	uartReadFn  UARTReadFn
	uartWriteFn UARTWriteFn

	// Interface
	interfaceName string

	// Whether or not a reset is required
	resetRequired bool

	// I2C instance state
	i2cAddress uint16
}

// Report a critical card error
func (context *Context) cardReportError(err error) {
	if context.Debug {
		fmt.Printf("*** %s\n", err)
	}
}

// DebugOutput enables/disables debug output
func (context *Context) DebugOutput(enabled bool) (wasEnabled bool) {
	wasEnabled = context.Debug
	context.Debug = enabled
	return
}

// Identify the type of this Notecard connection
func (context *Context) Identify() (name string) {
	return context.interfaceName
}

// Reset serial to a known state
func cardResetSerial(context *Context) (err error) {

	// In order to ensure that we're not getting the reply to a failed
	// transaction from a prior session, drain any pending input prior
	// to transmitting a command.  Note that we use this technique of
	// looking for a known reply to \n, rather than just "draining
	// anything pending on serial", because the nature of read() is
	// that it blocks (until timeout) if there's nothing available.
	var length int
	buf := make([]byte, 2048)
	for {
		_, err = context.uartWriteFn([]byte("\n"))
		if err != nil {
			err = fmt.Errorf("error transmitting to module: %s %s", err, ErrCardIo)
			context.cardReportError(err)
			return
		}
		time.Sleep(750 * time.Millisecond)
		length, err = context.uartReadFn(buf)
		if err != nil {
			err = fmt.Errorf("error reading from module: %s %s", err, ErrCardIo)
			context.cardReportError(err)
			return
		}
		somethingFound := false
		nonCRLFFound := false
		for i := 0; i < length && !nonCRLFFound; i++ {
			if false {
				fmt.Printf("chr: 0x%02x '%c'\n", buf[i], buf[i])
			}
			if buf[i] != '\r' {
				somethingFound = true
				if buf[i] != '\n' {
					nonCRLFFound = true
				}
			}
		}
		if somethingFound && !nonCRLFFound {
			break
		}
	}

	// Done
	return

}

// OpenUART opens the card on the specified uart
func OpenUART(uartReadFn UARTReadFn, uartWriteFn UARTWriteFn) (context *Context, err error) {

	// Create the context structure
	context = &Context{}
	context.Debug = InitialDebugMode
	context.interfaceName = "uart"

	// Set up I/O functions
	context.uartReadFn = uartReadFn
	context.uartWriteFn = uartWriteFn

	// Set up class functions
	context.CloseFn = cardCloseSerial
	context.ResetFn = cardResetSerial
	context.TransactionFn = cardTransactionSerial

	// Done
	return

}

// Reset I2C to a known good state
func cardResetI2C(context *Context) (err error) {

	// Synchronize by guaranteeing not only that I2C works, but that we drain the remainder of any
	// pending partial reply from a previously-aborted session.
	chunklen := 0
	for {

		// Read the next chunk of available data
		_, available, err2 := context.i2cReadBytes(chunklen)
		if err2 != nil {
			err = fmt.Errorf("error reading chunk: %s %s", err2, ErrCardIo)
			return
		}

		// If nothing left, we're ready to transmit a command to receive the data
		if available == 0 {
			break
		}

		// For the next iteration, reaad the min of what's available and what we're permitted to read
		chunklen = available
		if chunklen > CardI2CMax {
			chunklen = CardI2CMax
		}

	}

	// Done
	return

}

// OpenI2C opens the card on I2C
func OpenI2C(addr uint16, i2cTxFn I2CTxFn) (context *Context, err error) {

	// Create the context structure
	context = &Context{}
	context.Debug = InitialDebugMode
	context.interfaceName = "i2c"
	if addr == 0 {
		context.i2cAddress = DefaultI2CAddress
	} else {
		context.i2cAddress = addr
	}

	// Set up I/O functions
	context.i2cTxFn = i2cTxFn

	// Set up class functions
	context.CloseFn = cardCloseI2C
	context.ResetFn = cardResetI2C
	context.TransactionFn = cardTransactionI2C

	// Done
	return

}

// WriteBytes writes a buffer to I2C
// By design, must not send more than once every 1Ms
func (context *Context) i2cWriteBytes(buf []byte) (err error) {
	time.Sleep(1 * time.Millisecond)
	reg := make([]byte, 1)
	reg[0] = byte(len(buf))
	reg = append(reg, buf...)
	err = context.i2cTxFn(context.i2cAddress, reg, nil)
	if err != nil {
		err = fmt.Errorf("i2c write: %s", err)
	}
	return
}

// ReadBytes reads a buffer from I2C and returns how many are still pending
// By design, must not send more than once every 1Ms
func (context *Context) i2cReadBytes(datalen int) (outbuf []byte, available int, err error) {
	time.Sleep(1 * time.Millisecond)
	readbuf := make([]byte, datalen+2)
	// Retry, for robustness
	for i := 0; ; i++ {
		reg := make([]byte, 2)
		reg[0] = byte(0)
		reg[1] = byte(datalen)
		err = context.i2cTxFn(context.i2cAddress, reg, readbuf)
		if err == nil {
			break
		}
		if i >= 10 {
			err = fmt.Errorf("i2c read: %s", err)
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	if len(readbuf) < 2 {
		err = fmt.Errorf("i2c read: not enough data (%d < 2)", len(readbuf))
		return
	}
	available = int(readbuf[0])
	if available > 253 {
		err = fmt.Errorf("i2c read: available too large (%d >253)", available)
		return
	}
	good := readbuf[1]
	if len(readbuf) < int(2+good) {
		err = fmt.Errorf("i2c read: insufficient data (%d < %d)", len(readbuf), 2+good)
		return
	}
	if 2 > 2+good {
		if false {
			fmt.Printf("i2c read(%d): %v\n", datalen, readbuf)
		}
		err = fmt.Errorf("i2c read: %d bytes returned while expecting %d", good, datalen)
		return
	}
	outbuf = readbuf[2 : 2+good]
	return
}

// Reset the port
func (context *Context) Reset() (err error) {
	context.resetRequired = false
	return context.ResetFn(context)
}

// Close the port
func (context *Context) Close() {
	context.CloseFn(context)
}

// Close serial
func cardCloseSerial(context *Context) {
}

// Close I2C
func cardCloseI2C(context *Context) {
}

// NewRequest creates a new request that is guaranteed to get a response
// from the Notecard.  Note that this method is provided merely as syntactic sugar, as of the form
// req := tinynote.NewRequest("note.add")
func NewRequest(reqType string) (req map[string]interface{}) {
	return map[string]interface{}{
		"req": reqType,
	}
}

// NewCommand creates a new command that requires no response from the notecard.
func NewCommand(reqType string) (cmd map[string]interface{}) {
	return map[string]interface{}{
		"cmd": reqType,
	}
}

// NewBody creates a new body.  Note that this method is provided
// merely as syntactic sugar, as of the form
// body := note.NewBody()
func NewBody() (body map[string]interface{}) {
	return make(map[string]interface{})
}

// Request performs a card transaction with a JSON structure and doesn't return a response
// (This is for semantic compatibility with other languages.)
func (context *Context) Request(req map[string]interface{}) (err error) {
	_, err = context.Transaction(req)
	return
}

// RequestResponse performs a card transaction with a JSON structure and doesn't return a response
// (This is for semantic compatibility with other languages.)
func (context *Context) RequestResponse(req map[string]interface{}) (rsp map[string]interface{}, err error) {
	return context.Transaction(req)
}

// Response is used in rare cases where there is a transaction that returns multiple responses
func (context *Context) Response() (rsp map[string]interface{}, err error) {
	return context.Transaction(nil)
}

// Transaction performs a card transaction with a JSON structure
func (context *Context) Transaction(req map[string]interface{}) (rsp map[string]interface{}, err error) {

	// Handle the special case where we are just processing a response
	var reqJSON []byte
	if req == nil {

		reqJSON = []byte("")

	} else {

		// Marshal the request to JSON
		reqJSON, _ = ObjectToJSON(req)

	}

	// Perform the transaction
	rspJSON, err2 := context.TransactionJSON(reqJSON)
	if err2 != nil {
		err = fmt.Errorf("error from TransactionJSON: %s", err2)
		return
	}

	// Unmarshal for convenience of the caller
	rsp, err = JSONToObject(rspJSON)
	if err != nil {
		err = fmt.Errorf("error unmarshaling reply from module: %s %s", err, ErrCardIo)
		return
	}

	// Done
	return
}

// TransactionJSON performs a card transaction using raw JSON []bytes
func (context *Context) TransactionJSON(reqJSON []byte) (rspJSON []byte, err error) {

	// Unmarshal the request to peek inside it.  Also, accept a zero-length request as a valid case
	// because we use this in the test fixture where  we just accept pure responses w/o requests.
	var req map[string]interface{}
	var noResponseRequested bool

	// Make sure that it is valid JSON, because the transports won't validate this
	// and they may misbehave if they do not get a valid JSON response back.
	req, err = JSONToObject(reqJSON)
	if err != nil {
		return
	}

	// If this is a hub.set, generate a user agent object if one hasn't already been supplied
	if !context.DisableUA && (req["req"] == "hub.set" || req["cmd"] == "hub.set") && req["body"] == nil {
		ua := context.UserAgent()
		if ua != nil {
			req["body"] = ua
			reqJSON, _ = ObjectToJSON(req)
		}
	}

	// Determine whether or not a response will be expected from the notecard by
	// examining the req and cmd fields
	noResponseRequested = req["req"] == "" && req["cmd"] != ""

	// Make sure that the JSON has a single \n terminator
	for {
		if strings.HasSuffix(string(reqJSON), "\n") {
			reqJSON = []byte(strings.TrimSuffix(string(reqJSON), "\n"))
			continue
		}
		if strings.HasSuffix(string(reqJSON), "\r") {
			reqJSON = []byte(strings.TrimSuffix(string(reqJSON), "\r"))
			continue
		}
		break
	}
	reqJSON = []byte(string(reqJSON) + "\n")

	// Debug
	if context.Debug {
		var j []byte
		j, _ = ObjectToJSON(req)
		fmt.Printf("%s\n", string(j))
	}

	// Only one caller at a time accessing the I/O port
	transLock.Lock()

	// Do a reset if one was pending
	if context.resetRequired {
		context.Reset()
	}

	// Perform the transaction
	rspJSON, err = context.TransactionFn(context, noResponseRequested, reqJSON)
	if err != nil {
		context.resetRequired = true
	}

	// If this was a card restore, we want to hold everyone back if we reset the card
	if req["req"] == "card.restore" || req["req"] == "card.restart" {
		time.Sleep(8 * time.Second)
	}
	transLock.Unlock()

	// If no response, we're done
	if noResponseRequested {
		rspJSON = []byte("{}")
		return
	}

	// Decode the response to create an error if the transaction returned an error.  We
	// do this because it's SUPER inconvenient to always be checking for a response error
	// vs an error on the transaction itself
	rsp := map[string]interface{}{}
	if err == nil {
		rsp, err = JSONToObject(rspJSON)
	}
	if IsError(err, rsp) {
		if req["req"] == "" {
			err = fmt.Errorf("%s", ErrorString(err, rsp))
		} else {
			err = fmt.Errorf("%s: %s", req["req"], ErrorString(err, rsp))
		}
	}

	// Debug
	if context.Debug {
		fmt.Printf("%s", string(rspJSON))
	}

	// Done
	return

}

// Perform a card transaction over serial under the assumption that request already has '\n' terminator
func cardTransactionSerial(context *Context, noResponse bool, reqJSON []byte) (rspJSON []byte, err error) {

	// Initialize timing parameters
	if RequestSegmentMaxLen < 0 {
		RequestSegmentMaxLen = CardRequestSerialSegmentMaxLen
	}
	if RequestSegmentDelayMs < 0 {
		RequestSegmentDelayMs = CardRequestSerialSegmentDelayMs
	}

	// Handle the special case where we are looking only for a reply
	if len(reqJSON) > 0 {

		// Transmit the request in segments so as not to overwhelm the notecard's interrupt buffers
		segOff := 0
		segLeft := len(reqJSON)
		for {
			segLen := segLeft
			if segLen > RequestSegmentMaxLen {
				segLen = RequestSegmentMaxLen
			}
			_, err = context.uartWriteFn(reqJSON[segOff : segOff+segLen])
			if err != nil {
				err = fmt.Errorf("error transmitting to module: %s %s", err, ErrCardIo)
				context.cardReportError(err)
				return
			}
			segOff += segLen
			segLeft -= segLen
			if segLeft == 0 {
				break
			}
			time.Sleep(time.Duration(RequestSegmentDelayMs) * time.Millisecond)
		}

	}

	// If no response, we're done
	if noResponse {
		return
	}

	// Read the reply until we get '\n' at the end
	waitBeganSecs := time.Now().Unix()
	for {
		var length int
		buf := make([]byte, 2048)
		length, err = context.uartReadFn(buf)
		if err != nil {
			if err == io.EOF {
				// Just a read timeout
				continue
			}
			// Ignore [flaky] hardware errors for up to several seconds
			if (time.Now().Unix() - waitBeganSecs) > 2 {
				err = fmt.Errorf("error reading from module: %s %s", err, ErrCardIo)
				context.cardReportError(err)
				return
			}
			time.Sleep(1 * time.Second)
			continue
		}
		rspJSON = append(rspJSON, buf[:length]...)
		if strings.HasSuffix(string(rspJSON), "\n") {
			break
		}
	}

	// Done
	return

}

// Perform a card transaction over I2C under the assumption that request already has '\n' terminator
func cardTransactionI2C(context *Context, noResponse bool, reqJSON []byte) (rspJSON []byte, err error) {

	// Initialize timing parameters
	if RequestSegmentMaxLen < 0 {
		RequestSegmentMaxLen = CardRequestI2CSegmentMaxLen
	}
	if RequestSegmentDelayMs < 0 {
		RequestSegmentDelayMs = CardRequestI2CSegmentDelayMs
	}

	// Transmit the request in chunks, but also in segments so as not to overwhelm the notecard's interrupt buffers
	chunkoffset := 0
	jsonbufLen := len(reqJSON)
	sentInSegment := 0
	for jsonbufLen > 0 {
		chunklen := CardI2CMax
		if jsonbufLen < chunklen {
			chunklen = jsonbufLen
		}
		err = context.i2cWriteBytes(reqJSON[chunkoffset : chunkoffset+chunklen])
		if err != nil {
			err = fmt.Errorf("write error: %s %s", err, ErrCardIo)
			return
		}
		chunkoffset += chunklen
		jsonbufLen -= chunklen
		sentInSegment += chunklen
		if sentInSegment > RequestSegmentMaxLen {
			sentInSegment = 0
			time.Sleep(time.Duration(RequestSegmentDelayMs) * time.Millisecond)
		}
		time.Sleep(time.Duration(RequestSegmentDelayMs) * time.Millisecond)
	}

	// If no response, we're done
	if noResponse {
		return
	}

	// Loop, building a reply buffer out of received chunks.  We'll build the reply in the same
	// buffer we used to transmit, and will grow it as necessary.
	jsonbufLen = 0
	receivedNewline := false
	chunklen := 0
	expireSecs := 60
	expires := time.Now().Add(time.Duration(expireSecs) * time.Second)
	for {

		// Read the next chunk
		readbuf, available, err2 := context.i2cReadBytes(chunklen)
		if err2 != nil {
			err = fmt.Errorf("read error: %s %s", err2, ErrCardIo)
			return
		}

		// Append to the JSON being accumulated
		rspJSON = append(rspJSON, readbuf...)
		readlen := len(readbuf)
		jsonbufLen += readlen

		// If we received something, reset the expiration
		if readlen > 0 {
			expires = time.Now().Add(time.Duration(90) * time.Second)
		}

		// If the last byte of the chunk is \n, chances are that we're done.  However, just so
		// that we pull everything pending from the module, we only exit when we've received
		// a newline AND there's nothing left available from the module.
		if readlen > 0 && readbuf[readlen-1] == '\n' {
			receivedNewline = true
		}

		// For the next iteration, reaad the min of what's available and what we're permitted to read
		chunklen = available
		if chunklen > CardI2CMax {
			chunklen = CardI2CMax
		}

		// If there's something available on the notecard for us to receive, do it
		if chunklen > 0 {
			continue
		}

		// If there's nothing available and we received a newline, we're done
		if receivedNewline {
			break
		}

		// If we've timed out and nothing's available, exit
		expired := false
		timeoutSecs := 0
		if jsonbufLen == 0 {
			expired = time.Now().After(expires)
			timeoutSecs = expireSecs
		}
		if expired {
			err = fmt.Errorf("transaction timeout (received %d bytes in %d secs) %s", jsonbufLen, timeoutSecs, ErrCardIo+ErrTimeout)
			return
		}

	}

	// Done
	return
}

// IsError tests to see if a response contains an error
func IsError(err error, rsp map[string]interface{}) bool {
	if err != nil {
		return true
	}
	if rsp == nil {
		return false
	}
	if rsp["err"] == nil {
		return false
	}
	if rsp["err"] == "" {
		return false
	}
	return true
}

// ErrorString returns the error within a response
func ErrorString(err error, rsp map[string]interface{}) string {
	if err != nil {
		return fmt.Sprintf("%s", err)
	}
	if !IsError(err, rsp) {
		return ""
	}
	return rsp["err"].(string)
}

// ErrorContains tests to see if an error contains an error keyword that we might expect
func ErrorContains(err error, errKeyword string) bool {
	if err == nil {
		return false
	}
	return strings.Contains(fmt.Sprintf("%s", err), errKeyword)
}

// ErrorClean removes all error keywords from an error string
func ErrorClean(err error) error {
	errstr := fmt.Sprintf("%s", err)
	for {
		left := strings.SplitN(errstr, "{", 2)
		if len(left) == 1 {
			break
		}
		errstr = left[0]
		b := strings.SplitN(left[1], "}", 2)
		if len(b) > 1 {
			errstr += strings.TrimPrefix(b[1], " ")
		}
	}
	return fmt.Errorf(errstr)
}

// ErrorJSON returns a JSON object with nothing but an error code, and with an optional message
func ErrorJSON(message string, err error) (rspJSON []byte) {
	if message == "" {
		rspJSON = []byte(fmt.Sprintf("{\"err\":\"%q\"}", err))
	} else if err == nil {
		rspJSON = []byte(fmt.Sprintf("{\"err\":\"%q\"}", message))
	} else {
		rspJSON = []byte(fmt.Sprintf("{\"err\":\"%q: %q\"}", message, err))
	}
	return
}

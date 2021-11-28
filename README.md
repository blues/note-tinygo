# [Blues Wireless][blues]

The note-tinygo Go library for communicating with Blues Wireless Notecard via serial or I²C.

This library allows you to control a Notecard by coding in Go for the TinyGo platform.
Your program may configure Notecard and send Notes to [Notehub.io][notehub].

See also:
* [note-c][note-c] for C bindings
* [note-python][note-python] for Python bindings

## Installing
For all releases, we have compiled the notecard utility for different OS and architectures [here](https://github.com/blues/note-go/releases).
If you don't see your OS and architecture supported, please file an issue and we'll add it to new releases.

[blues]: https://blues.com
[notehub]: https://notehub.io
[note-arduino]: https://github.com/blues/note-arduino
[note-c]: https://github.com/blues/note-c
[note-go]: https://github.com/blues/note-go
[note-tinygo]: https://github.com/blues/note-tinygo
[note-python]: https://github.com/blues/note-python

## Dependencies
- Install tinygo and the tinygo tools [(here)](https://tinygo.org/getting-started/install/)

## Example
```golang
// Example that uses the cellular Blues Wireless Notecard to send data to the notehub.io
// routing service, then routing it through to your own service in JSON on a REST endpoint.

package main

import (
	"fmt"
	"machine"
	"time"

	tinynote "github.com/blues/note-tinygo"
)

// Your Notehub project's ProductUID (so the notecard knows where to send its data)
const productUID = "net.ozzie.ray:test"

// Tinygo's main program
func main() {

	// Use default configuration of I2C
	machine.I2C0.Configure(machine.I2CConfig{})

	// Create a function for this machine type that performs I2C I/O
	i2cTxFn := func(addr uint16, wb []byte, rb []byte) (err error) {
		return machine.I2C0.Tx(addr, wb, rb)
	}

	// Open an I2C channel to the Notecard, supplying the I2C I/O function
	notecard, err := tinynote.OpenI2C(tinynote.DefaultI2CAddress, i2cTxFn)
	if err != nil {
		fmt.Printf("error opening notecard i2c port: %s\n", err)
		return
	}

	// Enable trace output so we can visualize requests/responses
	notecard.DebugOutput(true)

	// Configure the Notecard and set it to auto-provision to your project
	req := tinynote.NewRequest("hub.set")
	req["product"] = productUID // which notehub project we're using
	req["mode"] = "continuous"  // stay online continuously
	req["outbound"] = 60        // how often (mins) to auto-sync if pending data
	err = notecard.Request(req)
	if err != nil {
		fmt.Printf("%s: %s\n", req["req"], err)
	}

	// Enter a loop that sends data to the Notehub repeatedly. Data on the notecard
	// is stored within a user-defined JSON 'body', carried within an envelope called
	// a 'note' that is automatically tagged with time and location metadata.
	for i := 0; ; i++ {
		time.Sleep(time.Second * 10)

		req := tinynote.NewRequest("note.add") // 'add a note' transaction

		// Create 'body' by using Golang's standard container for a JSON data
		// structure, which is a map of fieldname-indexed data of any type.
		// This 'body' is completely user-defined, and would presumably contain
		// you sensor data.
		body := map[string]interface{}{}
		body["test1"] = i
		body["test2"] = float64(i) + float64(i)*0.2
		body["test3"] = fmt.Sprintf("0x%04x", i)
		body["test5"] = (i & 1) == 0
		body["test6"] = map[string]interface{}{"hello": "world"}

		req["body"] = body // add the user-defined body to the note
		req["sync"] = true // for this test, sync to notehub immediately

		// Send request to notecard and check the response for errors
		rsp, err := notecard.RequestResponse(req)
		if tinynote.IsError(err, rsp) {
			fmt.Printf("%s: %s\n", req["req"], tinynote.ErrorString(err, rsp))
			continue
		}

	}

}

```

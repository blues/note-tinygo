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
package main

import (
    "fmt"
    "machine"
    "time"

    tinynote "github.com/blues/note-tinygo"
)

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

    // Disable Notecard sync                                                                                            
    req := tinynote.NewRequest("hub.set")
    req["mode"] = "off"
    err = notecard.Request(req)
    if err != nil {
        fmt.Printf("%s: %s\n", req["req"], err)
    }

    // Enter a loop that performs a harmless Notecard transaction                                                       
    for {
        time.Sleep(time.Second * 2)
        req := tinynote.NewRequest("card.version")
        rsp, err := notecard.RequestResponse(req)
        if tinynote.IsError(err, rsp) {
            fmt.Printf("%s: %s\n", req["req"], tinynote.ErrorString(err, rsp))
            continue
        }
    }

}

```

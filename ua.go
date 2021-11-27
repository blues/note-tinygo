// Copyright 2017 Blues Inc.  All rights reserved.
// Use of this source code is governed by licenses granted by the
// copyright holder including that found in the LICENSE file.

package tinynote

import (
	"fmt"
	"runtime"
)

// UserAgent is for Someday when the machine package supports finding the
// characteristics of the machine, this is the place where we'd provide it.
func (context *Context) UserAgent() (ua map[string]interface{}) {

	ua = map[string]interface{}{}
	ua["agent"] = "note-tinygo"
	ua["compiler"] = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	ua["req_interface"] = context.interfaceName

	return

}

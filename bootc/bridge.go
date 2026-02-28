/*
   Copyright 2026 Sumicare

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package bootc

/*
#cgo LDFLAGS: -L${SRCDIR}/../target/release -lbootc_bridge -lm -ldl -lpthread
#cgo pkg-config: ostree-1 glib-2.0 gobject-2.0 gio-2.0 openssl zlib libzstd

#include <stdlib.h>
#include <stdint.h>

extern int32_t bootc_run(int32_t argc, const char *const *argv);
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

//go:generate cargo build --release -p bootc-bridge

var ErrBootcExit = errors.New("bootc exited with error")

// BootcRun invokes bootc via the Rust bridge. Not concurrency-safe.
func BootcRun(args []string) error {
	argc := C.int32_t(len(args))
	argv := make([]*C.char, 0, len(args))
	for _, a := range args {
		argv = append(argv, C.CString(a))
	}
	defer func() {
		for _, p := range argv {
			C.free(unsafe.Pointer(p))
		}
	}()

	rc := C.bootc_run(argc, &argv[0])

	if rc != 0 {
		return fmt.Errorf("%w: code %d", ErrBootcExit, rc)
	}

	return nil
}

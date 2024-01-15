// Copyright 2023 Hoster Authors. All rights reserved.
// Use of this source code is governed by an Apache License 2.0
// license that can be found in the LICENSE file.

package HosterLogger

import (
	"fmt"
)

// Test Info Func
func (l *Log) Spacer() {
	if l.Term {
		fmt.Println()
	}
}

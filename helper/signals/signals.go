// Copyright IBM Corp. 2015, 2026
// SPDX-License-Identifier: MPL-2.0

package signals

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"
)

var SIGNULL os.Signal = syscall.Signal(0)
var ValidSignals []string
var MonitoredSignals []os.Signal

func init() {
	for k, v := range SignalLookup {
		ValidSignals = append(ValidSignals, k)
		MonitoredSignals = append(MonitoredSignals, v)
	}
	sort.Strings(ValidSignals)
}

func Parse(s string) (os.Signal, error) {
	sig, ok := SignalLookup[strings.ToUpper(s)]
	if !ok {
		return nil, fmt.Errorf("invalid signal %q - valid signals are %q", s, ValidSignals)
	}
	return sig, nil
}

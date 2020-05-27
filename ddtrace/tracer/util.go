// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package tracer

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	// expLine matches a line in the /proc/self/cgroup file. It has a submatch for the last element (path), which contains the container ID.
	expLine = regexp.MustCompile(`^\d+:[^:]*:(.+)$`)
	// expContainerID matches contained IDs and sources. Source: https://github.com/Qard/container-info/blob/master/index.js
	expContainerID = regexp.MustCompile(`([0-9a-f]{8}[-_][0-9a-f]{4}[-_][0-9a-f]{4}[-_][0-9a-f]{4}[-_][0-9a-f]{12}|[0-9a-f]{64})(?:.scope)?$`)
)

// toFloat64 attempts to convert value into a float64. If the value is an integer
// greater or equal to 2^53 or less than or equal to -2^53, it will not be converted
// into a float64 to avoid losing precision. If it succeeds in converting, toFloat64
// returns the value and true, otherwise 0 and false.
func toFloat64(value interface{}) (f float64, ok bool) {
	const max = (int64(1) << 53) - 1
	const min = -max
	switch i := value.(type) {
	case byte:
		return float64(i), true
	case float32:
		return float64(i), true
	case float64:
		return i, true
	case int:
		return float64(i), true
	case int16:
		return float64(i), true
	case int32:
		return float64(i), true
	case int64:
		if i > max || i < min {
			return 0, false
		}
		return float64(i), true
	case uint:
		return float64(i), true
	case uint16:
		return float64(i), true
	case uint32:
		return float64(i), true
	case uint64:
		if i > uint64(max) {
			return 0, false
		}
		return float64(i), true
	default:
		return 0, false
	}
}

// parseUint64 parses a uint64 from either an unsigned 64 bit base-10 string
// or a signed 64 bit base-10 string representing an unsigned integer
func parseUint64(str string) (uint64, error) {
	if strings.HasPrefix(str, "-") {
		id, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return 0, err
		}
		return uint64(id), nil
	}
	return strconv.ParseUint(str, 10, 64)
}

func FindContainerID() string {
	f, err := os.Open("/proc/self/cgroup")
	if err == nil {
		defer f.Close()
		scn := bufio.NewScanner(f)
		for scn.Scan() {
			path := expLine.FindStringSubmatch(scn.Text())
			if len(path) != 2 {
				// invalid entry, continue
				continue
			}
			if id := expContainerID.FindString(path[1]); id != "" {
				return id
			}
		}
	}
	return ""
}

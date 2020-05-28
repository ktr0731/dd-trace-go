// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package internal

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"testing"
)

var (
	// expLine matches a line in the /proc/self/cgroup file. It has a submatch for the last element (path), which contains the container ID.
	expLine = regexp.MustCompile(`^\d+:[^:]*:(.+)$`)
	// expContainerID matches contained IDs and sources. Source: https://github.com/Qard/container-info/blob/master/index.js
	expContainerID = regexp.MustCompile(`([0-9a-f]{8}[-_][0-9a-f]{4}[-_][0-9a-f]{4}[-_][0-9a-f]{4}[-_][0-9a-f]{12}|[0-9a-f]{64})(?:.scope)?$`)
	// cgroupPath is the path to the cgroup file where we can find the container id if one exists.
	cgroupPath = "/proc/self/cgroup"
)

func FindContainerID() string {
	f, err := os.Open(cgroupPath)
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

func testOverrideCgroup(t *testing.T, in string) func() {
	origCgroupPath := cgroupPath

	tmpFile, err := ioutil.TempFile(os.TempDir(), "fake-cgroup-")
	if err != nil {
		t.Fatalf("failed to create fake cgroup file: %v", err)
	}
	cgroupPath = tmpFile.Name()
	_, err = io.WriteString(tmpFile, in)
	if err != nil {
		t.Fatalf("failed writing to fake cgroup file: %v", err)
	}
	err = tmpFile.Close()
	if err != nil {
		t.Fatalf("failed closing fake cgroup file: %v", err)
	}

	return func() {
		os.Remove(tmpFile.Name())
		cgroupPath = origCgroupPath
	}
}

// TestOverrideContainerID is a utility to be used by tests to allow overriding detected container id. It returns a
// function that will revert the original behaviour and is meant to be called through defer.
func TestFakeContainerID(t *testing.T) (string, func()) {
	containerIdBytes := make([]byte, 32)
	rand.Read(containerIdBytes)
	containerId := fmt.Sprintf("%x", containerIdBytes)
	return containerId, testOverrideCgroup(t,
		fmt.Sprintf("10:hugetlb:/kubepods/burstable/podfd52ef25-a87d-11e9-9423-0800271a638e/%s", containerId))
}

// TestOverrideContainerID is a utility to be used by tests to allow overriding detected container id. It returns a
// function that will revert the original behaviour and is meant to be called through defer.
func TestEmptyContainerID(t *testing.T) func() {
	return testOverrideCgroup(t, "")
}

// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	klog "k8s.io/klog/v2"
)

// GetRootDir returns the root directory of this git repo.
func GetRootDir() (string, error) {
	_, s, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("could not determine Caller to get root path")
	}
	t := "test" + string(os.PathSeparator)
	i := strings.Index(s, t)
	if i < 0 {
		return "", fmt.Errorf("could not determine Caller to get root path")
	}
	return s[:i], nil
}

// GetRootDirOrDie returns the root directory of this git repo or dies.
func GetRootDirOrDie() string {
	rootDir, err := GetRootDir()
	if err != nil {
		klog.Fatal(err)
	}
	return rootDir
}

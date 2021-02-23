//  Copyright 2021 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//
// +build integration

package main

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"testing"
)

func TestAddLocalRoute(t *testing.T) {
	var tests = []struct {
		route     string
		ipversion int
	}{
		{"23.34.45.56/18", 4},
		{"2600:1901:ffb0:721d:8000::/96", 6},
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Errorf("error populating interfaces: %v", err)
	}
	var iface net.Interface
	for _, iface := range interfaces {
		if iface.Name != "lo" {
			break
		}
	}

	for _, tt := range tests {
		if err := addLocalRoute(tt.route, iface.Name); err != nil {
			t.Errorf("error calling addLocalRoute: %v", err)
		}
		args := fmt.Sprintf("route -%d list table local %s scope host proto 66", tt.ipversion, tt.route)
		res := runCmdOutput(exec.Command("ip", strings.Split(args, " ")...))
		if res.ExitCode() != 0 {
			t.Errorf("error confirming route was added: %v", err)
		}

		if res.Stdout() != tt.route {
			t.Errorf("route output does not match expectation: got %v, expected %v", res.Stdout(), tt.route)
		}
	}

}

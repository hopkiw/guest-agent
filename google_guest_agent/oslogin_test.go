//  Copyright 2019 Google Inc. All Rights Reserved.
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

package main

import (
	"strings"
	"testing"
)

func TestFilterGoogleLines(t *testing.T) {
	var tests = []struct {
		contents, want []string
	}{
		{
			[]string{
				"line1",
				"line2",
				googleComment,
				"line3 after google comment",
				"line4",
				googleBlockStart,
				"line5 inside google block",
				"line6 inside google block",
				googleBlockEnd,
				"line7",
			},
			[]string{
				"line1",
				"line2",
				"line4",
				"line7",
			},
		},
		{
			[]string{
				"line1",
				"line2",
				googleBlockEnd,
				"line3",
				"line4",
			},
			[]string{
				"line1",
				"line2",
				"line3",
				"line4",
			},
		},
		{
			[]string{
				googleBlockStart,
				"line1 inside google block",
				"line2 inside google block",
				googleBlockEnd,
				"line3",
			},
			[]string{
				"line3",
			},
		},
		{
			[]string{
				googleBlockStart,
				"line1 inside google block",
				googleBlockStart,
				"line2 inside google block",
				googleBlockEnd,
				"line3",
				googleBlockEnd,
				"line4",
			},
			[]string{
				"line3",
				"line4",
			},
		},
		{
			[]string{
				googleBlockEnd,
				googleBlockStart,
				"line1 inside google block",
				"line2 inside google block",
				googleComment,
				googleBlockEnd,
				"line3",
			},
			[]string{
				"line3",
			},
		},
	}

	cmpslice := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for idx := 0; idx < len(a); idx++ {
			if a[idx] != b[idx] {
				return false
			}
		}
		return true
	}

	for idx, tt := range tests {
		if res := filterGoogleLines(strings.Join(tt.contents, "\n")); !cmpslice(res, tt.want) {
			t.Errorf("test %v: want: %v, got: %v\n", idx, tt.want, res)
		}
	}
}

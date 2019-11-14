//  Copyright 2017 Google Inc. All Rights Reserved.
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
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func TestGetWantedArgs(t *testing.T) {
	getWantedTests := []struct {
		arg  string
		os   string
		want []string
	}{
		{
			"specialize",
			"windows",
			[]string{
				"sysprep-specialize-script-bat",
				"sysprep-specialize-script-cmd",
				"sysprep-specialize-script-ps1",
				"sysprep-specialize-script-url",
			},
		},
		{
			"startup",
			"windows",
			[]string{
				"windows-startup-script-bat",
				"windows-startup-script-cmd",
				"windows-startup-script-ps1",
				"windows-startup-script-url",
			},
		},
		{
			"shutdown",
			"windows",
			[]string{
				"windows-shutdown-script-bat",
				"windows-shutdown-script-cmd",
				"windows-shutdown-script-ps1",
				"windows-shutdown-script-url",
			},
		},
		{
			"startup",
			"linux",
			[]string{
				"startup-script-url",
				"startup-script",
			},
		},
		{
			"shutdown",
			"linux",
			[]string{
				"shutdown-script-url",
				"shutdown-script",
			},
		},
	}

	for _, tt := range getWantedTests {
		got, err := getWantedKeys([]string{"", tt.arg}, tt.os)
		if err != nil {
			t.Fatalf("validateArgs returned error: %v", err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("returned slice does not match expected one: got %v, want %v", got, tt.want)
		}
		_, err := getWantedKeys([]string{""})
		if err == nil {
			t.Errorf("0 args should produce an error")
		}
		_, err := getWantedKeys([]string{"", "", ""})
		if err == nil {
			t.Errorf("3 args should produce an error")
		}
	}
}

func TestGetExistingKeys(t *testing.T) {
	wantedKeys := []string{
		"sysprep-specialize-script-cmd",
		"sysprep-specialize-script-ps1",
		"sysprep-specialize-script-bat",
		"sysprep-specialize-script-url",
	}
	md := map[string]string{
		"sysprep-specialize-script-cmd": "cmd",
		"startup-script-cmd":            "cmd",
		"shutdown-script-ps1":           "ps1",
		"sysprep-specialize-script-url": "url",
		"sysprep-specialize-script-ps1": "ps1",
		"key":                           "value",
		"sysprep-specialize-script-bat": "bat",
	}
	want := map[string]string{
		"sysprep-specialize-script-cmd": "cmd",
		"sysprep-specialize-script-ps1": "ps1",
		"sysprep-specialize-script-bat": "bat",
		"sysprep-specialize-script-url": "url",
	}
	got := parseMetadata(md, wantedKeys)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parsed metadata does not match expectation, got: %v, want: %v", got, want)
	}
}

func TestFindMatch(t *testing.T) {
	matchTests := []struct {
		path, bucket, object string
	}{
		{"gs://bucket/object", "bucket", "object"},
		{"gs://bucket/some/object", "bucket", "some/object"},
		{"http://bucket.storage.googleapis.com/object", "bucket", "object"},
		{"https://bucket.storage.googleapis.com/object", "bucket", "object"},
		{"https://bucket.storage.googleapis.com/some/object", "bucket", "some/object"},
		{"http://storage.googleapis.com/bucket/object", "bucket", "object"},
		{"http://commondatastorage.googleapis.com/bucket/object", "bucket", "object"},
		{"https://storage.googleapis.com/bucket/object", "bucket", "object"},
		{"https://commondatastorage.googleapis.com/bucket/object", "bucket", "object"},
		{"https://storage.googleapis.com/bucket/some/object", "bucket", "some/object"},
		{"https://commondatastorage.googleapis.com/bucket/some/object", "bucket", "some/object"},
	}

	for _, tt := range matchTests {
		bucket, object := findMatch(tt.path)
		if bucket != tt.bucket {
			t.Errorf("returned bucket does not match expected one for %q:\n  got %q, want: %q", tt.path, bucket, tt.bucket)
		}
		if object != tt.object {
			t.Errorf("returned object does not match expected one for %q\n:  got %q, want: %q", tt.path, object, tt.object)
		}
	}
}

func TestGetMetadata(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"key1":"value1","key2":"value2"}`)
	}))
	defer ts.Close()

	metadataURL = ts.URL
	metadataHang = ""

	want := map[string]string{"key1": "value1", "key2": "value2"}
	got, err := getMetadataAttributes("")
	if err != nil {
		t.Fatalf("error running getMetadataAttributes: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("metadata does not match expectation, got: %q, want: %q", got, want)
	}
}

func TestGetScripts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.String() == "/instance/attributes?instance" {
			fmt.Fprintln(w, `{"test":"instance"}`)
		} else if r.URL.String() == "/project/attributes?project" {
			fmt.Fprintln(w, `{"test":"project"}`)
		} else if r.URL.String() == "/instance/attributes?project" {
			fmt.Fprintln(w, `{"some-metadata":"instance"}`)
		} else if r.URL.String() == "/instance/attributes?both" {
			fmt.Fprintln(w, `{"test":"instance"}`)
		} else if r.URL.String() == "/project/attributes?both" {
			fmt.Fprintln(w, `{"test":"project"}`)
		} else {
			fmt.Fprintln(w, "{}")
		}
	}))
	defer ts.Close()
	metadataURL = ts.URL

	mdsm := map[metadataScriptType]string{
		cmd: "test",
	}

	tests := []struct {
		desc, hang, script string
	}{
		{"just instance", "?instance", "instance"},
		{"just project", "?project", "project"},
		{"instance overrides project", "?both", "instance"},
	}

	for _, tt := range tests {
		metadataHang = tt.hang
		msdd, err := getScripts(mdsm)
		if err != nil {
			t.Fatalf("%s error: %v", tt.desc, err)
		}

		if len(msdd) != 1 {
			t.Errorf("%s len(msdd) != 1", tt.desc)
		}
		if msdd[0].Script != tt.script {
			t.Errorf("%s Script (%q) != %q", tt.desc, msdd[0].Script, tt.script)
		}
		if msdd[0].Metadata != "test" {
			t.Errorf(`%s Metadata (%q) != "test"`, tt.desc, msdd[0].Metadata)
		}
	}
}

func TestPrepareURIExec(t *testing.T) {
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.String()
		m := r.Method

		if strings.Contains(u, "dne") {
			w.WriteHeader(http.StatusNotFound)
		} else if m == "GET" && strings.Contains(u, "test") {
			fmt.Fprint(w, "test")
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "testGCSClient unknown request: %+v\n", r)
		}
	}))

	var err error
	testStorageClient, err = storage.NewClient(ctx, option.WithEndpoint(ts.URL), option.WithHTTPClient(&http.Client{Timeout: 1 * time.Second}))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc, url, wExt, want string
		cmd                   *exec.Cmd
		err                   bool
	}{
		{"url dne", ts.URL + "/dne", "", "", nil, true},
		{"url ok ps1", ts.URL + "/test.ps1", ".ps1", "test", exec.Command("powershell.exe", powerShellArgs...), false},
		{"url ok cmd", ts.URL + "/test.cmd", ".cmd", "test", nil, false},
		{"url ok bat", ts.URL + "/test.bat?some=thing", ".bat", "test", nil, false},
		//{"gs url ok", "gs://foo/test.cmd?some=thing", ".cmd", "test", nil, false},
		{"bad ext", ts.URL + "/test.bad", "", "", nil, true},
	}

	name := "foo"
	for _, tt := range tests {
		c, dir, err := prepareURIExec(ctx, &metadataScript{Script: tt.url, Metadata: name})
		if dir != "" {
			defer os.RemoveAll(dir)
		}
		if err != nil && !tt.err {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
		} else if err == nil && tt.err {
			t.Errorf("%s: expected error but got nil", tt.desc)
		} else if err != nil && tt.err {
			continue
		} else {
			path := filepath.Join(dir, name+tt.wExt)
			data, err := ioutil.ReadFile(path)
			if err != nil {
				t.Errorf("%s: error reading tmp file: %v", tt.desc, err)
			}
			if string(data) != tt.want {
				t.Errorf("%s: content does not match, got: %q, want: %q", tt.desc, string(data), tt.want)
			}
			if tt.cmd != nil {
				tt.cmd.Args = append(tt.cmd.Args, path)
			} else {
				tt.cmd = exec.Command(path)
			}
			if !reflect.DeepEqual(c, tt.cmd) {
				t.Errorf("%s: exec does not match:\n got: %+v\n want: %+v", tt.desc, c, tt.cmd)
			}
		}
	}
}

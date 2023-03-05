package main

import (
	"fmt"
	"os/user"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/go-cmp/cmp"
)

func TestTargetFromConnString(t *testing.T) {
	u, err := user.Current()
	if err != nil {
		panic(err)
	}
	username := u.Username
	tests := map[string]struct {
		input    string
		expected Target
	}{
		"empty string": {
			input: "",
			expected: Target{
				Host: "/tmp",
				Port: 5432,
				User: username,
			},
		},
		"hostname": {
			input: "db.example.com",
			expected: Target{
				Host: "db.example.com",
				Port: 5432,
				User: username,
			},
		},
		"hostname:port": {
			input: "db.example.com:4567",
			expected: Target{
				Host: "db.example.com",
				Port: 4567,
				User: username,
			},
		},
		"username@hostname": {
			input: "user@db.example.com",
			expected: Target{
				Host: "db.example.com",
				Port: 5432,
				User: "user",
			},
		},
		"username@hostname:port": {
			input: "user@db.example.com:4567",
			expected: Target{
				Host: "db.example.com",
				Port: 4567,
				User: "user",
			},
		},
		"postgres://hostname": {
			input: "postgres://db.example.com",
			expected: Target{
				Host: "db.example.com",
				Port: 5432,
				User: username,
			},
		},
	}
	for desc, tc := range tests {
		tg := Target{}
		err := tg.FromConnString(tc.input)
		if err != nil {
			t.Fatalf("%s: error %v", desc, err)
		}
		diff := cmp.Diff(tc.expected, tg)
		if diff != "" {
			t.Errorf("%s: mismatch:\n%s", desc, diff)
		}
	}
}

func TestTargetFromFlags(t *testing.T) {
	tests := map[string]struct {
		set      func()
		expected Target
	}{
		"none": {
			set:      func() {},
			expected: Target{},
		},
		"hostname": {
			set: func() {
				pgHost = ptr.String("db.example.com")
			},
			expected: Target{
				Host: "db.example.com",
			},
		},
		"hostname and port": {
			set: func() {
				pgHost = ptr.String("db.example.com")
				pgPort = ptr.String("4567")
			},
			expected: Target{
				Host: "db.example.com",
				Port: 4567,
			},
		},
		"username and hostname": {
			set: func() {
				pgUser = ptr.String("user")
				pgHost = ptr.String("db.example.com")
			},
			expected: Target{
				User: "user",
				Host: "db.example.com",
			},
		},
		"database": {
			set: func() {
				pgDatabase = ptr.String("example")
			},
			expected: Target{
				Database: "example",
			},
		},
		"app name": {
			set: func() {
				pgAppName = ptr.String("myapp")
			},
			expected: Target{
				AppName: "myapp",
			},
		},
	}
	for desc, tc := range tests {
		pgHost = nil
		pgPort = nil
		pgDatabase = nil
		pgUser = nil
		pgPassword = nil
		pgAppName = nil
		tg := Target{}
		tc.set()
		err := tg.FromFlags()
		if err != nil {
			t.Fatalf("%s: error %v", desc, err)
		}
		diff := cmp.Diff(tc.expected, tg)
		if diff != "" {
			t.Errorf("%s: mismatch:\n%s", desc, diff)
		}
	}
}

func TestTargetToConnConfig(t *testing.T) {
	defaultAppName := "pgping/" + VERSION
	tests := map[string]struct {
		input    Target
		expected string
	}{
		"empty": {
			input:    Target{},
			expected: fmt.Sprintf("postgres://?application_name=%s", defaultAppName),
		},
		"host": {
			input: Target{
				Host: "db.example.com",
			},
			expected: fmt.Sprintf("postgres://db.example.com?application_name=%s", defaultAppName),
		},
		"host and port": {
			input: Target{
				Host: "db.example.com",
				Port: 4567,
			},
			expected: fmt.Sprintf("postgres://db.example.com:4567?application_name=%s", defaultAppName),
		},
		"username and host": {
			input: Target{
				User: "user",
				Host: "db.example.com",
			},
			expected: fmt.Sprintf("postgres://user@db.example.com?application_name=%s", defaultAppName),
		},
		"username and password and host": {
			input: Target{
				User:     "user",
				Password: "hunter2",
				Host:     "db.example.com",
			},
			expected: fmt.Sprintf("postgres://user:hunter2@db.example.com?application_name=%s", defaultAppName),
		},
		"username and password": {
			input: Target{
				User:     "user",
				Password: "hunter2",
			},
			expected: fmt.Sprintf("postgres://user:hunter2@?application_name=%s", defaultAppName),
		},
		"database": {
			input: Target{
				Database: "test",
			},
			expected: fmt.Sprintf("postgres:///test?application_name=%s", defaultAppName),
		},
		"database and host": {
			input: Target{
				Host:     "db.example.com",
				Database: "test",
			},
			expected: fmt.Sprintf("postgres://db.example.com/test?application_name=%s", defaultAppName),
		},
		"app name": {
			input: Target{
				AppName: "CustomApp",
			},
			expected: "postgres://?application_name=CustomApp",
		},
	}
	for desc, tc := range tests {
		connConfig, err := tc.input.ToConnConfig()
		if err != nil {
			t.Fatalf("%s: error %v", desc, err)
		}
		diff := cmp.Diff(tc.expected, connConfig.ConnString())
		if diff != "" {
			t.Errorf("%s: mismatch:\n%s", desc, diff)
		}
	}
}

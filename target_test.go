package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
)

func TestTargetFromConnString(t *testing.T) {
	// reference empty config from pgx
	ref, err := pgx.ParseConfig("postgres://")
	if err != nil {
		panic(err)
	}
	tests := map[string]struct {
		input    string
		initial  Target
		expected Target
	}{
		"empty string": {
			input: "",
			expected: Target{
				Host: ref.Host,
				Port: int(ref.Port),
				User: ref.User,
			},
		},
		"hostname": {
			input: "db.example.com",
			expected: Target{
				Host: "db.example.com",
				Port: 5432,
				User: ref.User,
			},
		},
		"hostname:port": {
			input: "db.example.com:4567",
			expected: Target{
				Host: "db.example.com",
				Port: 4567,
				User: ref.User,
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
				User: ref.User,
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

func TestTargetFromNetrc(t *testing.T) {
	tests := map[string]struct {
		initial  Target
		netrc    string
		expected Target
	}{
		"empty": {
			initial:  Target{},
			netrc:    "",
			expected: Target{},
		},
		"non-matching line": {
			initial: Target{
				Host: "db.example.com",
			},
			netrc: "machine example.com login daniel password qwerty",
			expected: Target{
				Host: "db.example.com",
			},
		},
		"matching line": {
			initial: Target{
				Host: "db.example.com",
			},
			netrc: "machine db.example.com login daniel password qwerty",
			expected: Target{
				Host:     "db.example.com",
				User:     "daniel",
				Password: "qwerty",
			},
		},
		"non-overriding": {
			initial: Target{
				Host:     "db.example.com",
				User:     "user",
				Password: "hunter2",
			},
			netrc: "machine db.example.com login daniel password qwerty",
			expected: Target{
				Host:     "db.example.com",
				User:     "user",
				Password: "hunter2",
			},
		},
	}
	for desc, tc := range tests {
		tg := tc.initial
		file, err := os.CreateTemp("", "*.netrc")
		if err != nil {
			t.Fatalf("%s: error creating temp file %v", desc, err)
		}
		defer os.Remove(file.Name())
		file.Write([]byte(tc.netrc))
		err = file.Close()
		if err != nil {
			t.Fatalf("%s: error closing temp file %v", desc, err)
		}
		err = tg.FromNetrc(file.Name())
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

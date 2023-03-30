package main

import (
	"os"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

func ReferenceConnConfig(t *testing.T) *pgx.ConnConfig {
	t.Helper()
	// reference empty config from pgx
	ref, err := pgx.ParseConfig("postgres://")
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func TestTargetFromEnv(t *testing.T) {
	tests := map[string]struct {
		env      map[string]string
		initial  Target
		expected Target
	}{
		"empty": {
			env:      map[string]string{},
			expected: Target{},
		},
		"hostname": {
			env: map[string]string{
				"PGHOST": "db.example.com",
			},
			expected: Target{
				Host: "db.example.com",
			},
		},
		"hostname and port": {
			env: map[string]string{
				"PGHOST": "db.example.com",
				"PGPORT": "4567",
			},
			expected: Target{
				Host: "db.example.com",
				Port: 4567,
			},
		},
		"username and hostname": {
			env: map[string]string{
				"PGUSER": "user",
				"PGHOST": "db.example.com",
			},
			expected: Target{
				User: "user",
				Host: "db.example.com",
			},
		},
		"username and password": {
			env: map[string]string{
				"PGUSER":     "user",
				"PGPASSWORD": "hunter2",
			},
			expected: Target{
				User:     "user",
				Password: "hunter2",
			},
		},
		"database": {
			env: map[string]string{
				"PGDATABASE": "example",
			},
			expected: Target{
				Database: "example",
			},
		},
		"app name": {
			env: map[string]string{
				"PGAPPNAME": "myapp",
			},
			expected: Target{
				AppName: "myapp",
			},
		},
	}
	for desc, tc := range tests {
		tg := tc.initial
		getenv := func(key string) string {
			if v, ok := tc.env[key]; ok {
				return v
			}
			return ""
		}
		err := tg.FromEnv(getenv)
		if err != nil {
			t.Fatalf("%s: error %v", desc, err)
		}
		diff := cmp.Diff(tc.expected, tg)
		if diff != "" {
			t.Errorf("%s: mismatch:\n%s", desc, diff)
		}
	}
}

func TestTargetFromConnString(t *testing.T) {
	ref := ReferenceConnConfig(t)
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
		tg := tc.initial
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
		initial  Target
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
		"username and password": {
			set: func() {
				pgUser = ptr.String("user")
				pgPassword = ptr.String("hunter2")
			},
			expected: Target{
				User:     "user",
				Password: "hunter2",
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
		tg := tc.initial
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

type expectedConnConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	AppName  string
}

func (ecc *expectedConnConfig) GetHost(t *testing.T) string {
	if ecc.Host != "" {
		return ecc.Host
	}
	return ReferenceConnConfig(t).Host
}

func (ecc *expectedConnConfig) GetPort(t *testing.T) int {
	t.Helper()
	if ecc.Port != 0 {
		return ecc.Port
	}

	return int(ReferenceConnConfig(t).Port)
}

func (ecc *expectedConnConfig) GetUser(t *testing.T) string {
	t.Helper()
	if ecc.User != "" {
		return ecc.User
	}

	return ReferenceConnConfig(t).User
}

func (ecc *expectedConnConfig) GetAppName(t *testing.T) string {
	t.Helper()
	defaultAppName := "pgping/" + VERSION
	if ecc.AppName == "" {
		return defaultAppName
	}
	return ecc.AppName
}

func TestTargetToConnConfig(t *testing.T) {

	tests := map[string]struct {
		input    Target
		expected expectedConnConfig
	}{
		"empty": {
			input:    Target{},
			expected: expectedConnConfig{},
		},
		"host": {
			input: Target{
				Host: "db.example.com",
			},
			expected: expectedConnConfig{
				Host: "db.example.com",
			},
		},
		"host and port": {
			input: Target{
				Host: "db.example.com",
				Port: 4567,
			},
			expected: expectedConnConfig{
				Host: "db.example.com",
				Port: 4567,
			},
		},
		"username and host": {
			input: Target{
				User: "user",
				Host: "db.example.com",
			},
			expected: expectedConnConfig{
				User: "user",
				Host: "db.example.com",
			},
		},
		"username and password and host": {
			input: Target{
				User:     "user",
				Password: "hunter2",
				Host:     "db.example.com",
			},
			expected: expectedConnConfig{
				User:     "user",
				Password: "hunter2",
				Host:     "db.example.com",
			},
		},
		"username and password": {
			input: Target{
				User:     "user",
				Password: "hunter2",
			},
			expected: expectedConnConfig{
				User:     "user",
				Password: "hunter2",
			},
		},
		"database": {
			input: Target{
				Database: "test",
			},
			expected: expectedConnConfig{
				Database: "test",
			},
		},
		"database and host": {
			input: Target{
				Host:     "db.example.com",
				Database: "test",
			},
			expected: expectedConnConfig{
				Host:     "db.example.com",
				Database: "test",
			},
		},
		"app name": {
			input: Target{
				AppName: "CustomApp",
			},
			expected: expectedConnConfig{
				AppName: "CustomApp",
			},
		},
	}
	for desc, tc := range tests {
		tc := tc
		t.Run(desc, func(t *testing.T) {
			connConfig, err := tc.input.ToConnConfig()
			if err != nil {
				t.Fatalf("%s: error %v", desc, err)
			}
			assert.Equal(t, tc.expected.GetAppName(t), connConfig.RuntimeParams["application_name"])
			assert.Equal(t, tc.expected.Database, connConfig.Database)
			assert.Equal(t, tc.expected.GetHost(t), connConfig.Host)
			assert.Equal(t, tc.expected.Password, connConfig.Password)
			assert.EqualValues(t, tc.expected.GetPort(t), connConfig.Port)
			assert.Equal(t, tc.expected.GetUser(t), connConfig.User)
		})
	}
}

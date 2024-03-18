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
		"don't overwrite set values from empty connstring": {
			input: "",
			initial: Target{
				Host: "db.example.com",
				Port: 4567,
				User: "user",
			},
			expected: Target{
				Host: "db.example.com",
				Port: 4567,
				User: "user",
			},
		},
		"don't overwrite set value from connstring that doesn't set them": {
			input: "user2:hunter2@db2.example.com",
			initial: Target{
				Host: "db.example.com",
				Port: 4567,
				User: "user",
			},
			expected: Target{
				Host:     "db2.example.com",
				Port:     4567,
				User:     "user2",
				Password: "hunter2",
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
		_, err = file.WriteString(tc.netrc)
		if err != nil {
			t.Fatalf("%s: error writing temp file %v", desc, err)
		}
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
	t.Helper()
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
	t.Parallel()

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
		"unparsable password": {
			input: Target{
				User:     "user",
				Password: "rdspostgres.123456789012.us-west-2.rds.amazonaws.com:5432/?Action=connect&DBUser=user&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=ASIAXXXXXXXXXXXXXXXX%2F20230330%2Fus-west-2%2Frds-db%2Faws4_request&X-Amz-Date=20230330T154114Z&X-Amz-Expires=900&X-Amz-SignedHeaders=host&X-Amz-Security-Token=8QvrlkjXuPwM2Dif69QtSqtMftfDUfxSZ60OADWHKmjF3Nmu9Y0LSzvBbE3jRqhaF07q+mv9pa8TDqVwg4Ic6OW5dw8UMo+pOcsu26hmhMowUOv95/GACY6zKDGaM3x3SCVavKcSFa8bzCTlBIS0O/JMWaDZHumrb+Nxne5zWlpEKgEEXvXq6jV2gSF2pQmnxThMPWJCTLMRsWuUvyOTaztF/y4Dx92U9BQVWpvXg0gWBj6eVWeDHjRINYQzlVFrReF71MREjBs0Do9ijFCEjbqwsHmD5NUnbz1yX+bTttNuVGG3Kdl3Nea+11/FbBYa1JS7tIt8naCMZ29MO9wlW2vu0vE29bEI0ESgiO9RR/XBhJy1U783CA9oNVExIeQ6l3l9wFrLBLkQISp8WpXd1a0YEWA4FnCTP++Ox5ogHF1RVFHx1VohE/VrGHWJamudgs2lmV453Yg2cAWzQ+XxDk838zIDIgYhPc6QWlVqDsLAU04r7FSYIla2JmNVMLNQM+pIpo4ROGxF4B8/lMSqP6b8Vcd+xh428eGVoOf0K8ZwtQ/aTeLoEOUvJuA9ztCqjPhu8/QLqEXI3pS9UCeJ17mUJpeju22du/v/IFeExQTSopER/FPV5VbvAHiS+sur35d34wtcibKSS1d1HIz0i8Dttgk7K+WStkeyVLBaqbsvo6xAGAkx8rLk2fCMmthPyYvI18U1UV/fuUo5WyXXdm79uJ1ZuubzB1u1RX0sa72YFVFgBKklyQuBukVeGHETHh7no5ghnXWQhjymud4tGxVuCzVEceZKSXhO63qe13uGwDMQ9BMVV5YDfNblHzqFimQzoJE/okCVS5gATf3ny6PS/pK40N9HP0hXTu6F6+QqS9Gc4MnkNOFLyghv+oMQ1Mh2EMglzBYdNdM8RWDfGv/xUIYyRodLvZNJlQ7edb5JMeEHe8CuD+2SovMkKbohUbKhyQkZRxAa3bilk3MU25RJVV70H+L9haxR5v1dsaHbHY&X-Amz-Signature=qQ8hqnAbW6TyOimaVOT0FJYqGQ2LO8iCaxT/UFcFHMpi/9XA7kAXcL0aPwpImubN", //nolint:lll
				Host:     "rdspostgres.123456789012.us-west-2.rds.amazonaws.com",
			},
			expected: expectedConnConfig{
				User:     "user",
				Password: "rdspostgres.123456789012.us-west-2.rds.amazonaws.com:5432/?Action=connect&DBUser=user&X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=ASIAXXXXXXXXXXXXXXXX%2F20230330%2Fus-west-2%2Frds-db%2Faws4_request&X-Amz-Date=20230330T154114Z&X-Amz-Expires=900&X-Amz-SignedHeaders=host&X-Amz-Security-Token=8QvrlkjXuPwM2Dif69QtSqtMftfDUfxSZ60OADWHKmjF3Nmu9Y0LSzvBbE3jRqhaF07q+mv9pa8TDqVwg4Ic6OW5dw8UMo+pOcsu26hmhMowUOv95/GACY6zKDGaM3x3SCVavKcSFa8bzCTlBIS0O/JMWaDZHumrb+Nxne5zWlpEKgEEXvXq6jV2gSF2pQmnxThMPWJCTLMRsWuUvyOTaztF/y4Dx92U9BQVWpvXg0gWBj6eVWeDHjRINYQzlVFrReF71MREjBs0Do9ijFCEjbqwsHmD5NUnbz1yX+bTttNuVGG3Kdl3Nea+11/FbBYa1JS7tIt8naCMZ29MO9wlW2vu0vE29bEI0ESgiO9RR/XBhJy1U783CA9oNVExIeQ6l3l9wFrLBLkQISp8WpXd1a0YEWA4FnCTP++Ox5ogHF1RVFHx1VohE/VrGHWJamudgs2lmV453Yg2cAWzQ+XxDk838zIDIgYhPc6QWlVqDsLAU04r7FSYIla2JmNVMLNQM+pIpo4ROGxF4B8/lMSqP6b8Vcd+xh428eGVoOf0K8ZwtQ/aTeLoEOUvJuA9ztCqjPhu8/QLqEXI3pS9UCeJ17mUJpeju22du/v/IFeExQTSopER/FPV5VbvAHiS+sur35d34wtcibKSS1d1HIz0i8Dttgk7K+WStkeyVLBaqbsvo6xAGAkx8rLk2fCMmthPyYvI18U1UV/fuUo5WyXXdm79uJ1ZuubzB1u1RX0sa72YFVFgBKklyQuBukVeGHETHh7no5ghnXWQhjymud4tGxVuCzVEceZKSXhO63qe13uGwDMQ9BMVV5YDfNblHzqFimQzoJE/okCVS5gATf3ny6PS/pK40N9HP0hXTu6F6+QqS9Gc4MnkNOFLyghv+oMQ1Mh2EMglzBYdNdM8RWDfGv/xUIYyRodLvZNJlQ7edb5JMeEHe8CuD+2SovMkKbohUbKhyQkZRxAa3bilk3MU25RJVV70H+L9haxR5v1dsaHbHY&X-Amz-Signature=qQ8hqnAbW6TyOimaVOT0FJYqGQ2LO8iCaxT/UFcFHMpi/9XA7kAXcL0aPwpImubN", //nolint:lll
				Host:     "rdspostgres.123456789012.us-west-2.rds.amazonaws.com",
			},
		},
	}
	for desc, tc := range tests {
		tc := tc
		t.Run(desc, func(t *testing.T) {
			t.Parallel()
			connConfig, err := tc.input.ToConnConfig()
			if err != nil {
				t.Fatalf("error %v", err)
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

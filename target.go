package main

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jdxcode/netrc"
)

type Target struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	AppName  string
}

func (t *Target) FromConnString(s string) error {
	if !strings.Contains(s, "postgres://") {
		debugf("Target.FromConnString: adding connstring prefix to `%s`", s)
		s = "postgres://" + s
	}
	empty, err := pgx.ParseConfig("postgres://")
	if err != nil {
		return err
	}
	parsed, err := pgx.ParseConfig(s)
	if err != nil {
		return err
	}
	if parsed.Host != empty.Host || t.Host == "" {
		debugf("Target.FromConnString: setting host to `%s`", parsed.Host)
		t.Host = parsed.Host
	}
	if parsed.Port != empty.Port || t.Port == 0 {
		debugf("Target.FromConnString: setting host to `%d`", parsed.Port)
		t.Port = int(parsed.Port)
	}
	if parsed.Database != empty.Database || t.Database == "" {
		debugf("Target.FromConnString: setting database to `%s`", parsed.Database)
		t.Database = parsed.Database
	}
	if parsed.User != empty.User || t.User == "" {
		debugf("Target.FromConnString: setting user to `%s`", parsed.User)
		t.User = parsed.User
	}
	if parsed.Password != empty.Password || t.Password == "" {
		debugf("Target.FromConnString: setting password to `%s`", parsed.Password)
		t.Password = parsed.Password
	}
	return nil
}

func (t *Target) FromEnv(getenv func(string) string) error {
	if host := getenv("PGHOST"); host != "" {
		debugf("Target.FromEnv: setting host to `%s`", host)
		t.Host = host
	}
	if port := getenv("PGPORT"); port != "" {
		debugf("Target.FromEnv: setting port to `%s`", port)
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		t.Port = portInt
	}
	if database := getenv("PGDATABASE"); database != "" {
		debugf("Target.FromEnv: setting database to `%s`", database)
		t.Database = database
	}
	if user := getenv("PGUSER"); user != "" {
		debugf("Target.FromEnv: setting user to `%s`", user)
		t.User = user
	}
	if password := getenv("PGPASSWORD"); password != "" {
		debugf("Target.FromEnv: setting password to `%s`", password)
		t.Password = password
	}
	if appName := getenv("PGAPPNAME"); appName != "" {
		debugf("Target.FromEnv: setting appname to `%s`", appName)
		t.AppName = appName
	}
	return nil
}

func (t *Target) FromNetrc(path string) error {
	if path == "" {
		debugln("Target.FromNetrc: netrc path not provided, finding suitable netrc")
		if env := os.Getenv("NETRC"); env != "" {
			debugf("Target.FromNetrc: using $NETRC environment variable")
			path = env
		} else {
			debugf("Target.FromNetrc: $NETRC not set, using netrc in home directory")
			base := ".netrc"
			if runtime.GOOS == "windows" {
				base = "_netrc"
			}
			usr, err := user.Current()
			if err != nil {
				return err
			}
			path = filepath.Join(usr.HomeDir, base)
		}
	}
	debugf("Target.FromNetrc: using netrc at `%s`", path)
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		debugf("Target.FromNetrc: netrc at `%s` doesn't exist; skipping netrc configuration", path)
		return nil
	}
	if stat.IsDir() {
		debugf("Target.FromNetrc: netrc at `%s` is a directory; skipping netrc configuration", path)
		return nil
	}
	n, err := netrc.Parse(path)
	if err != nil {
		return err
	}
	machine := n.Machine(t.Host)
	if machine == nil {
		debugf("Target.FromNetrc: netrc doesn't contain a machine entry for `%s`; skipping netrc configuration", t.Host)
		return nil
	}
	empty, err := pgx.ParseConfig("postgres://")
	if err != nil {
		return err
	}
	if t.User == "" || t.User == empty.User {
		if username := machine.Get("login"); username != "" {
			debugf("Target.FromNetrc: setting user to `%s`", username)
			t.User = username
		}
	}
	if t.Password == "" {
		if password := machine.Get("password"); password != "" {
			debugf("Target.FromNetrc: setting password to `%s`", password)
			t.Password = password
		}
	}
	return nil
}

func (t *Target) FromFlags() error {
	if pgHost != nil && *pgHost != "" {
		debugf("Target.FromFlags: setting host to `%s`", *pgHost)
		t.Host = *pgHost
	}
	if pgPort != nil && *pgPort != "" {
		debugf("Target.FromFlags: setting port to `%s`", *pgPort)
		port, err := strconv.Atoi(*pgPort)
		if err != nil {
			return err
		}
		t.Port = port
	}
	if pgDatabase != nil && *pgDatabase != "" {
		debugf("Target.FromFlags: setting database to `%s`", *pgDatabase)
		t.Database = *pgDatabase
	}
	if pgUser != nil && *pgUser != "" {
		debugf("Target.FromFlags: setting user to `%s`", *pgUser)
		t.User = *pgUser
	}
	if pgPassword != nil && *pgPassword != "" {
		debugf("Target.FromFlags: setting password to `%s`", *pgPassword)
		t.Password = *pgPassword
	}
	if pgAppName != nil && *pgAppName != "" {
		debugf("Target.FromFlags: setting appname to `%s`", *pgAppName)
		t.AppName = *pgAppName
	}
	return nil
}

func (t *Target) ToConnConfig() (*pgx.ConnConfig, error) {
	var connString strings.Builder
	connString.WriteString("postgres://")
	if t.User != "" {
		connString.WriteString(t.User)
		connString.WriteString("@")
	}
	if t.Host != "" {
		connString.WriteString(t.Host)
	}
	if t.Port != 0 {
		connString.WriteString(":")
		connString.WriteString(strconv.Itoa(t.Port))
	}
	if t.Database != "" {
		connString.WriteString("/")
		connString.WriteString(t.Database)
	}
	connString.WriteString("?application_name=")
	if t.AppName == "" {
		connString.WriteString("pgping/" + VERSION)
	} else {
		connString.WriteString(t.AppName)
	}
	connConfig, err := pgx.ParseConfig(connString.String())
	if err != nil {
		return connConfig, err
	}
	if t.Password != "" {
		connConfig.Password = t.Password
	}
	return connConfig, nil
}

package main

import (
	"fmt"
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
		t.Host = parsed.Host
	}
	if parsed.Port != empty.Port || t.Port == 0 {
		t.Port = int(parsed.Port)
	}
	if parsed.Database != empty.Database || t.Database == "" {
		t.Database = parsed.Database
	}
	if parsed.User != empty.User || t.User == "" {
		t.User = parsed.User
	}
	if parsed.Password != empty.Password || t.Password == "" {
		t.Password = parsed.Password
	}
	return nil
}

func (t *Target) FromEnv(getenv func(string) string) error {
	if host := getenv("PGHOST"); host != "" {
		t.Host = host
	}
	if port := getenv("PGPORT"); port != "" {
		portInt, err := strconv.Atoi(port)
		if err != nil {
			return err
		}
		t.Port = portInt
	}
	if database := getenv("PGDATABASE"); database != "" {
		t.Database = database
	}
	if user := getenv("PGUSER"); user != "" {
		t.User = user
	}
	if password := getenv("PGPASSWORD"); password != "" {
		t.Password = password
	}
	if appName := getenv("PGAPPNAME"); appName != "" {
		t.AppName = appName
	}
	return nil
}

func (t *Target) FromNetrc(path string) error {
	if path == "" {
		if env := os.Getenv("NETRC"); env != "" {
			path = env
		} else {
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
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if stat.IsDir() {
		return nil
	}
	n, err := netrc.Parse(path)
	if err != nil {
		return err
	}
	machine := n.Machine(t.Host)
	if machine == nil {
		return nil
	}
	if t.User == "" {
		if username := machine.Get("login"); username != "" {
			t.User = username
		}
	}
	if t.Password == "" {
		if password := machine.Get("password"); password != "" {
			t.Password = password
		}
	}
	return nil
}

func (t *Target) FromFlags() error {
	if pgHost != nil && *pgHost != "" {
		t.Host = *pgHost
	}
	if pgPort != nil && *pgPort != "" {
		port, err := strconv.Atoi(*pgPort)
		if err != nil {
			return err
		}
		t.Port = port
	}
	if pgDatabase != nil && *pgDatabase != "" {
		t.Database = *pgDatabase
	}
	if pgUser != nil && *pgUser != "" {
		t.User = *pgUser
	}
	if pgPassword != nil && *pgPassword != "" {
		t.Password = *pgPassword
	}
	if pgAppName != nil && *pgAppName != "" {
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
		connString.WriteString(fmt.Sprint(t.Port))
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

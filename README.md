# pgping

Quick-n-dirty "ping" utility for testing connectivity to PostgreSQL databases

```
usage: pgping [<flags>] [<target>]

Flags:
  -h, --help                     Show context-sensitive help (also try --help-long and --help-man).
  -c, --count=-1                 stop after N pings
  -i, --wait=1s                  wait time between sending each ping
  -t, --timeout=5s               timeout for connections to the DB
      --query="SELECT 1"         Test query to execute on database
      --pg-host=PG-HOST
      --pg-port=PG-PORT
      --pg-database=PG-DATABASE
      --pg-user=PG-USER
      --pg-password=PG-PASSWORD
      --pg-app-name="pgping/0.3.0"


Args:
  [<target>]
```

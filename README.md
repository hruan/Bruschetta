# Bruschetta

Bruschetta is project I did for a course in [web application][1] (sorry,
Swedish only) at [Malmö University][2]. It's clobbered together with [Go][3]
and some JavaScript, mostly as a learning exercise for [Go][3].

## Prerequisites

- A working Go environment
- A PostgreSQL database, accepting TCP connections, with a table called
  `titles`; see `examples/table.sql` for column details
- Since Netflix no longer issue API keys, a bzip2 compressed catalog is
  assumed to be available at `DIR/bin/db/netflix.db.bz2`
- Rotten Tomatoes API credentials stored in `rt.json` and available in the
  same directory as executables; see `examples/rt.json`

## Setup

Bruschetta is assumed to reside in `DIR`.

1. Set `GOPATH` to `DIR`
2. Get dependencies: `go get cron/netflix bruschetta`
3. Build binaries: `go install cron/netflix bruschetta`
4. Run the `netflix` tool to populate database with Netflix catalog; use
   option `--help` see command line options
5. Run Bruschetta; again, use option `--help` for command line options
6. Point your browser to port 8888 on the host running Bruschetta

[1]: http://edu.mah.se/sv/Course/DA197A?v=1
[2]: http://www.mah.se/
[3]: http://golang.org/

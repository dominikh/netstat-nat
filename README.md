# netstat-nat

This is a reimplementation of the
[netstat-nat](http://tweegy.nl/projects/netstat-nat/) tool, written
entirely in Go. It uses the same command line flags and almost the
same output format so it can be used as a drop-in replacement in most
cases.

## Install

```sh
go get -u honnef.co/go/netstat-nat
```

## Usage

```sh
$GOPATH/bin/netstat-nat --help
```

## Differences

- The original version limits the printed hostnames to fixed width. We
  do not.

- The -x flag is a NOOP because we do not limit the length of
  hostnames.

- The -N flag is not yet supported.

package main

import (
	"flag"
	"time"
)

var httpAddr = flag.String("httpAddr", ":5000", "HTTP Address to listen to")
var requestCacheExpiration = flag.Duration("requestCache", 24*time.Hour, "Request cache expiration timeout")
var packageLruCache = flag.Int("packageLruCache", 10000, "Number of packages stored in memory")
var suites = flag.String("suites", "stretch,jessie,xenial,bionic", "A list of supported suites")

var parseDeb = flag.String("parseDeb", "", "Try to parse a debian archive")

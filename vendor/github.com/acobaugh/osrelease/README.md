# osrelease [![Build Status](https://travis-ci.org/cobaugh/osrelease.svg?branch=master)](https://travis-ci.org/cobaugh/osrelease)

A Go package to make reading in os-release files easy.

See https://www.freedesktop.org/software/systemd/man/os-release.html

## Installation
`$ go get github.com/cobaugh/osrelease`

## Usage

See [godoc](https://godoc.org/github.com/cobaugh/osrelease)

```golang
package main

import (
	"fmt"
	"github.com/cobaugh/osrelease"
)

func main() {
	// for reference, two variables are provided:
	fmt.Printf("EtcOsRelease = %v\n", osrelease.EtcOsRelease)
	fmt.Printf("UsrLibOsRelease = %v\n", osrelease.UsrLibOsRelease)

	// let osrelease find what file to load
	osrelease, err := osrelease.Read()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("PRETTY_NAME = %v\n", osrelease["PRETTY_NAME"])

	// specify the file to load explicitly
	osrelease, err = osrelease.ReadFile("/etc/os-release")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("PRETTY_NAME = %v\n", osrelease["PRETTY_NAME"])
}

```

Output:
```
$ ./examples 
EtcOsRelease = /etc/os-release
UsrLibOsRelease = /usr/lib/os-release
PRETTY_NAME = void
PRETTY_NAME = void```

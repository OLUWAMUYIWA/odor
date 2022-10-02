package main

import (
	"os"
)

func main() {
	driver := newDriver()
	if err := driver.Drive(); err != nil {
		os.Exit(1)
	}
}

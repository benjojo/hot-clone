// +build !linux

package main

import (
	"log"
	"os"
)

func setupBlkTrace(err error, f *os.File, eventConsumer chan string, deviceBaseName string) {
	log.Fatalf("Blktrace is a linux only feature")
}

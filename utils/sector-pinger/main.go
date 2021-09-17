package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {
	target := flag.String("device-to-destroy", "", "The device you want to utterly fuck up")
	S1 := flag.Int("sectorA", 1337, "First Sector you want to write to")
	S2 := flag.Int("sectorB", 2323, "Second Sector you want to write to")
	flag.Parse()

	tgt, err := os.OpenFile(*target, os.O_RDWR, 0777)
	if err != nil {
		log.Fatalf("%v", err)
	}

	for {
		time.Sleep(time.Millisecond * 500)
		Mark(*S1, tgt)
		time.Sleep(time.Millisecond * 500)
		Mark(*S2, tgt)
	}

}

var c = 0

func Mark(i int, tgt *os.File) {
	fmt.Printf("%d marked (%d bytes in)\n", i, i*512)
	_, err := tgt.Seek(int64(i*512), 0)
	if err != nil {
		log.Fatalf("%v", err)
	}
	_, err = tgt.Write([]byte(fmt.Sprintf(".%d ", c)))
	if err != nil {
		log.Fatalf("%v", err)
	}
	c++
}

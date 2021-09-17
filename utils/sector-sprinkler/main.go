package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	target := flag.String("device-to-destroy", "", "The device you want to utterly fuck up")
	spraySize := flag.Int("sectors", 1953125/10, "The amount of sectors you want to bound yourself to")
	flag.Parse()

	tgt, err := os.OpenFile(*target, os.O_RDWR, 0777)
	if err != nil {
		log.Fatalf("%v", err)
	}

	for i := 0; i < *spraySize; i++ {
		if i%7127 == 0 {
			fmt.Printf("%d marked (%d bytes in)\n", i, i*512)
			_, err = tgt.Seek(int64(i*512), 0)
			if err != nil {
				log.Fatalf("%v", err)
			}
			_, err = tgt.Write([]byte(fmt.Sprintf(".%d ", i)))
			if err != nil {
				log.Fatalf("%v", err)
			}
		}
	}
}

package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func main() {
	dev := flag.String("device", "", "The device you wise to hot-clone")
	flag.Parse()

	info := syscall.Sysinfo_t{}
	syscall.Sysinfo(&info)
	// AppStartTime := time.Now()
	// SystemStartTime := time.Now().Add((time.Second * time.Duration(info.Uptime)) * -1)
	deviceBaseName := filepath.Base(*dev)
	eventConsumer := make(chan unix.BLK_io_trace, 100)

	f, err := os.Open(*dev)
	if err != nil {
		log.Fatalf("cannot open device %v - %v", *dev, err)
	}

	diskSectorsCount := getTotalDeviceSectorsSize(deviceBaseName)
	setupBlkTrace(err, f, eventConsumer, deviceBaseName)

	DST := DirtySectorTracker{}
	DST.Setup(diskSectorsCount)

	go func() {
		for {
			time.Sleep(time.Second)
			log.Printf("Dirty %v sectors", DST.CountDirty())
		}
	}()

	for event := range eventConsumer {
		if event.Action&(1<<BLK_TC_WRITE) > 0 {
			log.Printf("Write: Sector %#v (%d bytes) | F: %x (%s) @%d (lag of %d)", event.Sector, event.Bytes, event.Action, unpackBits(event.Action), event.Time, event.Time-uint64(info.Uptime))
			if !(event.Sector == 0 && event.Bytes == 0) {
				DST.SetDirty(uint(event.Sector))
				otherSectors := (uint(event.Bytes) - 512) / 512
				for i := uint(0); i < otherSectors; i++ {
					DST.SetDirty(uint(event.Sector) + i)
				}
			}
		}
	}

	for {
		time.Sleep(time.Second)
		log.Printf("Debug time?")
	}

}

package main

import (
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

var DST DirtySectorTracker

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
	defer shutdownBlkTrace(f)

	DST = DirtySectorTracker{}
	DST.Setup(diskSectorsCount)

	go func() {
		for {
			time.Sleep(time.Second)
			DST.CountDirty()
			TotalRead := atomic.LoadInt64(&bytesRead)
			log.Printf("Read %v bytes -- Dirty %v sectors", TotalRead, DST.DirtySectors)
		}
	}()

	go trackEvents(eventConsumer, info)

	// Begin reading the block device
	BlockF, err := os.Open(*dev)
	if err != nil {
		log.Fatalf("cannot open block device %v - %v", *dev, err)
	}

	TotalRead := int64(0) // for use below only!!!
	for {
		data := make([]byte, 1024*1024)
		n, err := BlockF.Read(data)
		TotalRead += int64(n)
		atomic.StoreInt64(&bytesRead, TotalRead)
		if err == io.EOF {
			// we are done! time to image the other bits
			break
		} else if err != nil {
			log.Fatalf("Failed to read drive :/ -- %v", err)
			break
		}
		os.Stdout.Write(data)
	}

	// now let's catch up

}

var bytesRead int64

var dumpWrites = flag.Bool("print-writes", false, "print all writes happening")

func trackEvents(eventConsumer chan unix.BLK_io_trace, info syscall.Sysinfo_t) {
	for event := range eventConsumer {
		if event.Action&(1<<BLK_TC_WRITE) > 0 {
			if *dumpWrites {
				log.Printf("Write: Sector %#v (%d bytes) | F: %x (%s)", event.Sector, event.Bytes, event.Action, unpackBits(event.Action))
			}
			FarSide := (uint32(event.Sector) * 512) + event.Bytes
			ReadSoFar := atomic.LoadInt64(&bytesRead)

			if int64(FarSide) < (ReadSoFar - 1024*1000) {
				if !(event.Sector == 0 && event.Bytes == 0) {
					DST.SetDirty(uint(event.Sector))
					otherSectors := (uint(event.Bytes) - 512) / 512
					for i := uint(0); i < otherSectors; i++ {
						DST.SetDirty(uint(event.Sector) + i)
					}
				}
			}
		}
	}
}

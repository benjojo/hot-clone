package main

import (
	"flag"
	"fmt"
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
var debug = flag.Bool("verbose", false, "be extra verbose on whats happening")

func main() {
	dev := flag.String("device", "", "The device you wise to hot-clone")
	flag.Parse()

	// Nope, we are restoring instead
	if *reassemblePath != "" {
		reassembleMain()
		return
	}

	info := syscall.Sysinfo_t{}
	syscall.Sysinfo(&info)
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
			eventDrops := getBlkTraceDrops(deviceBaseName)
			log.Printf("Read %s -- %v Dirty sectors (%d event drops)", ByteCountIEC(TotalRead), DST.DirtySectors, eventDrops)
			if eventDrops != 0 {
				log.Fatalf("Event drops detected, cannot safely image device anymore")
			}
		}
	}()

	go trackEvents(eventConsumer, info)

	// Begin reading the block device
	BlockF, err := os.Open(*dev)
	if err != nil {
		log.Fatalf("cannot open block device %v - %v", *dev, err)
	}

	os.Stdout.WriteString("This-Is-A-Hot-Clone-Image See: https://github.com/benjojo/hot-clone\n")
	os.Stdout.WriteString(fmt.Sprintf("S:0\tL:%d\n", diskSectorsCount*512))
	TotalRead := int64(0) // for use below only!!!
	BytesLeftToRead := int(diskSectorsCount * 512)
	var buf []byte
	for {
		expectedRead := 1024 * 1024
		if BytesLeftToRead < 1024*1024 {
			expectedRead = BytesLeftToRead
		}

		if BytesLeftToRead == 0 {
			break
		}

		buf = make([]byte, expectedRead)

		n, err := BlockF.Read(buf)
		if err != nil {
			log.Fatalf("Disk read failure -- %v", err)
		}
		if n != expectedRead {
			log.Fatalf("Disk short read failure -- %v != %v (had %d bytes left)", n, expectedRead, BytesLeftToRead)
		}
		TotalRead += int64(n)
		atomic.StoreInt64(&bytesRead, TotalRead)
		BytesLeftToRead = BytesLeftToRead - expectedRead

		_, err = os.Stdout.Write(buf)
		if err != nil {
			log.Fatalf("Output file/device write failure -- %v", err)
		}
	}

	alreadyWarnedAboutOverread := false
	tmpSectors := diskSectorsCount
	// Attempt to over-read, in case the block device is actualy bigger
	for {
		data := make([]byte, 512)
		n, err := BlockF.Read(data)
		TotalRead += int64(n)
		atomic.StoreInt64(&bytesRead, TotalRead)
		if err == io.EOF {
			// we are done! time to image the other bits
			break
		} else if err != nil {
			break
		}

		if (uint64(bytesRead) > diskSectorsCount*512) && !alreadyWarnedAboutOverread {
			// Okay very interseting, the block layer let us read more data then there were sectors!
			alreadyWarnedAboutOverread = true
			log.Printf("Strange device! Let's us read more data than there are sectors!!!")
		}
		tmpSectors++
		os.Stdout.WriteString(fmt.Sprintf("S:%d\tL:%d\n", tmpSectors, n))
		os.Stdout.Write(data[:n])
		if n != 512 {
			// we are now out of alignment, time to leave
			log.Printf("*And* the strange device gave us a shorter than sector read()?!")
			break
		}
	}

	if alreadyWarnedAboutOverread {
		log.Printf("Device overread by %d sectors", tmpSectors-diskSectorsCount)
	}

	shutdownBlkTrace(f)
	// now let's catch up
	dirtySectorChannel := DST.GetDirtySectors()
	n := 0
	for sector := range dirtySectorChannel {
		BlockF.Seek(int64(sector)*512, 0)
		data := make([]byte, 512)
		br, _ := BlockF.Read(data)
		if br != 512 {
			log.Fatalf("Read for catchup failed!!! Only read %d bytes of a 512b sector", br)
		}
		os.Stdout.WriteString(fmt.Sprintf("S:%d\tL:%d\n", sector, 512))
		os.Stdout.Write(data)
		n++
		if n%(DST.DirtySectors/10) == 0 {
			log.Printf("Catching up %d/%d sectors", n, DST.DirtySectors)
		}
	}
	log.Printf("Done")
}

var bytesRead int64

var dumpWrites = flag.Bool("print-writes", false, "print all writes happening")

func trackEvents(eventConsumer chan unix.BLK_io_trace, info syscall.Sysinfo_t) {
	for event := range eventConsumer {
		if event.Action&(1<<BLK_TC_WRITE) > 0 {
			if *dumpWrites {
				log.Printf("Write: Sector %#v (%d) (%d bytes) | F: %x (%s)", event.Sector, event.Sector, event.Bytes, event.Action, unpackBits(event.Action))
			}
			FarSide := (uint32(event.Sector) * 512) + event.Bytes
			ReadSoFar := atomic.LoadInt64(&bytesRead)

			if int64(FarSide) < (ReadSoFar - 1024*1000) {
				if !(event.Sector == 0 && event.Bytes == 0) {
					DST.SetDirty(event.Sector)
					otherSectors := (uint(event.Bytes)) / 512
					for i := uint(0); i < otherSectors; i++ {
						DST.SetDirty(event.Sector + uint64(i))
					}
				}
			}
		} else {
			if *dumpWrites {
				log.Printf("????: Sector %#v (%d) (%d bytes) | F: %x (%s)", event.Sector, event.Sector, event.Bytes, event.Action, unpackBits(event.Action))
			}
		}
	}
}

func ByteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

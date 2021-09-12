package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

func main() {
	dev := flag.String("device", "", "The device you wise to hot-clone")
	flag.Parse()

	f, err := os.Open(*dev)
	if err != nil {
		log.Fatalf("cannot open device %v - %v", *dev, err)
	}

	// FD := f.Fd()
	// ioctl(3, BLKTRACESETUP, {act_mask=2, buf_size=524288, buf_nr=4, start_lba=0, end_lba=0, pid=0, name="vda"}) = 0
	traceOpts := unix.BLK_user_trace_setup{
		Act_mask: 2,
		Buf_size: 1024,
		Buf_nr:   4,
	}

	_, _, err = unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKTRACESETUP, uintptr(unsafe.Pointer(&traceOpts)))
	if err != nil {
		if err.Error() != "errno 0" {
			log.Fatalf("failed to BLKTRACESETUP -> %s", err)
			unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKTRACETEARDOWN, 0)

		}
	}

	_, _, err = unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKTRACESTART, 0)
	if err != nil {
		if err.Error() != "errno 0" {
			log.Fatalf("failed to BLKTRACESETUP -> %s", err)
			unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKTRACETEARDOWN, 0)
		}
	}
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		select {
		case sig := <-c:
			fmt.Printf("Got %s signal. Aborting...\n", sig)
			unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKTRACESTOP, 0)
			unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKTRACETEARDOWN, 0)
			os.Exit(1)
		}
	}()

	go readFiles(0)
	go readFiles(1)
	for {
		time.Sleep(time.Second)
		log.Printf("Debug time?")
	}

}

func readFiles(cpu int) {
	f, err := os.Open(fmt.Sprintf("/sys/kernel/debug/block/vda/trace%d", cpu))
	if err != nil {
		log.Fatalf("Cant open trace debugfs file?! %v", err)
	}

	for {
		data := make([]byte, 1024)
		n, err := f.Read(data)
		if err != nil {
			if err == io.EOF {
				time.Sleep(time.Millisecond * 10)
				continue
			}
			log.Printf("readFiles loop err %v", err)
			return
		}
		log.Printf("CPU %d read %d", cpu, n)
	}

}

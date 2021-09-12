// +build linux

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

func getTotalDeviceSectorsSize(deviceBaseName string) uint64 {
	b, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/block/%s/size", deviceBaseName))
	if err != nil {
		log.Fatalf("Cannot read device block size %v", err)
		return 0
	}

	i, err := strconv.ParseUint(strings.Trim(string(b), "\r\n\t "), 10, 64)
	if err != nil {
		log.Fatalf("Cannot parse device block size %v", err)
		return 0
	}

	return i
}

func setupBlkTrace(err error, f *os.File, eventConsumer chan unix.BLK_io_trace, deviceBaseName string) {
	traceOpts := unix.BLK_user_trace_setup{
		Act_mask: 2,
		Buf_size: 65536,
		Buf_nr:   4,
	}

	_, _, err = unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKTRACESETUP, uintptr(unsafe.Pointer(&traceOpts)))
	if err != nil {
		if err.Error() != "errno 0" {
			unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKTRACESTOP, 0)
			unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKTRACETEARDOWN, 0)
			log.Fatalf("failed to BLKTRACESETUP -> %s", err)
		}
	}

	_, _, err = unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKTRACESTART, 0)
	if err != nil {
		if err.Error() != "errno 0" {
			unix.Syscall(unix.SYS_IOCTL, f.Fd(), unix.BLKTRACETEARDOWN, 0)
			log.Fatalf("failed to BLKTRACESETUP -> %s", err)
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

	for i := 0; i < runtime.NumCPU(); i++ {
		go readBlkTraceEventFiles(i, eventConsumer, deviceBaseName)
	}
}

func readBlkTraceEventFiles(cpu int, out chan unix.BLK_io_trace, deviceBaseName string) {
	f, err := os.Open(fmt.Sprintf("/sys/kernel/debug/block/%s/trace%d", deviceBaseName, cpu))
	if err != nil {
		log.Fatalf("Cant open trace debugfs file?! %v", err)
	}

	for {
		BlkEvent := unix.BLK_io_trace{}
		err := binary.Read(f, binary.LittleEndian, &BlkEvent)

		if err != nil {
			if err == io.EOF {
				time.Sleep(time.Millisecond * 10)
				continue
			}
			log.Printf("readFiles loop err %v", err)
			return
		}

		data := make([]byte, BlkEvent.Len)
		_, err = f.Read(data)
		// log.Printf("CPU %d skipped %d", cpu, n)
		if err != nil {
			if err == io.EOF {
				time.Sleep(time.Millisecond * 10)
				continue
			}
			log.Printf("readFiles loop err %v", err)
			return
		}
		if BlkEvent.Error != 0 {
			if !(BlkEvent.Action&BLK_TA_UNPLUG_TIMER > 0) {
				log.Printf("Error !!!!!!!!!!!!!!!!!!!!! %#v", BlkEvent)
			}
		} else {
			// log.Printf("%#v", BlkEvent)
		}
		out <- BlkEvent
	}

}

const (
	BLK_TC_READ     = 1 << 0  /* reads */
	BLK_TC_WRITE    = 1 << 1  /* writes */
	BLK_TC_BARRIER  = 1 << 2  /* barrier */
	BLK_TC_SYNC     = 1 << 3  /* sync IO */
	BLK_TC_QUEUE    = 1 << 4  /* queueing/merging */
	BLK_TC_REQUEUE  = 1 << 5  /* requeueing */
	BLK_TC_ISSUE    = 1 << 6  /* issue */
	BLK_TC_COMPLETE = 1 << 7  /* completions */
	BLK_TC_FS       = 1 << 8  /* fs requests */
	BLK_TC_PC       = 1 << 9  /* pc requests */
	BLK_TC_NOTIFY   = 1 << 10 /* special message */
	BLK_TC_AHEAD    = 1 << 11 /* readahead */
	BLK_TC_META     = 1 << 12 /* metadata */
	BLK_TC_DISCARD  = 1 << 13 /* discard requests */
	BLK_TC_DRV_DATA = 1 << 14 /* binary per-driver data */
	BLK_TC_END      = 1 << 15 /* only 16-bits, reminder */

	BLK_TA_QUEUE        = 1 << 16 /* queued */
	BLK_TA_BACKMERGE    = 1 << 17 /* back merged to existing rq */
	BLK_TA_FRONTMERGE   = 1 << 18 /* front merge to existing rq */
	BLK_TA_GETRQ        = 1 << 19 /* allocated new request */
	BLK_TA_SLEEPRQ      = 1 << 20 /* sleeping on rq allocation */
	BLK_TA_REQUEUE      = 1 << 21 /* request requeued */
	BLK_TA_ISSUE        = 1 << 22 /* sent to driver */
	BLK_TA_COMPLETE     = 1 << 23 /* completed by driver */
	BLK_TA_PLUG         = 1 << 24 /* queue was plugged */
	BLK_TA_UNPLUG_IO    = 1 << 25 /* queue was unplugged by io */
	BLK_TA_UNPLUG_TIMER = 1 << 26 /* queue was unplugged by timer */
	BLK_TA_INSERT       = 1 << 27 /* insert request */
	BLK_TA_SPLIT        = 1 << 28 /* bio was split */
	BLK_TA_BOUNCE       = 1 << 29 /* bio was bounced */
	BLK_TA_REMAP        = 1 << 30 /* bio was remapped */
	BLK_TA_ABORT        = 1 << 31 /* request aborted */
	BLK_TA_DRV_DATA     = 1 << 32 /* driver-specific binary data */
)

var flagLetters = map[uint64]string{
	1 << 0:  "BLK_TC_READ",     /* reads */
	1 << 1:  "BLK_TC_WRITE",    /* writes */
	1 << 2:  "BLK_TC_BARRIER",  /* barrier */
	1 << 3:  "BLK_TC_SYNC",     /* sync IO */
	1 << 4:  "BLK_TC_QUEUE",    /* queueing/merging */
	1 << 5:  "BLK_TC_REQUEUE",  /* requeueing */
	1 << 6:  "BLK_TC_ISSUE",    /* issue */
	1 << 7:  "BLK_TC_COMPLETE", /* completions */
	1 << 8:  "BLK_TC_FS",       /* fs requests */
	1 << 9:  "BLK_TC_PC",       /* pc requests */
	1 << 10: "BLK_TC_NOTIFY",   /* special message */
	1 << 11: "BLK_TC_AHEAD",    /* readahead */
	1 << 12: "BLK_TC_META",     /* metadata */
	1 << 13: "BLK_TC_DISCARD",  /* discard requests */
	1 << 14: "BLK_TC_DRV_DATA", /* binary per-driver data */
	1 << 15: "BLK_TC_END",      /* only 16-bits, reminder */

	1 << 16: "BLK_TA_QUEUE",        /* queued */
	1 << 17: "BLK_TA_BACKMERGE",    /* back merged to existing rq */
	1 << 18: "BLK_TA_FRONTMERGE",   /* front merge to existing rq */
	1 << 19: "BLK_TA_GETRQ",        /* allocated new request */
	1 << 20: "BLK_TA_SLEEPRQ",      /* sleeping on rq allocation */
	1 << 21: "BLK_TA_REQUEUE",      /* request requeued */
	1 << 22: "BLK_TA_ISSUE",        /* sent to driver */
	1 << 23: "BLK_TA_COMPLETE",     /* completed by driver */
	1 << 24: "BLK_TA_PLUG",         /* queue was plugged */
	1 << 25: "BLK_TA_UNPLUG_IO",    /* queue was unplugged by io */
	1 << 26: "BLK_TA_UNPLUG_TIMER", /* queue was unplugged by timer */
	1 << 27: "BLK_TA_INSERT",       /* insert request */
	1 << 28: "BLK_TA_SPLIT",        /* bio was split */
	1 << 29: "BLK_TA_BOUNCE",       /* bio was bounced */
	1 << 30: "BLK_TA_REMAP",        /* bio was remapped */
	1 << 31: "BLK_TA_ABORT",        /* request aborted */
	// 1 << 32: "BLK_TA_DRV_DATA",     /* driver-specific binary data */
}

func unpackBits(in uint32) string {
	in2 := uint64(in)
	o := ""

	if in2&(1<<0) > 0 {
		o += " BLK_TC_READ"
	}
	if in2&(1<<1) > 0 {
		o += " BLK_TC_WRITE"
	}
	if in2&(1<<2) > 0 {
		o += " BLK_TC_BARRIER"
	}
	if in2&(1<<3) > 0 {
		o += " BLK_TC_SYNC"
	}
	if in2&(1<<4) > 0 {
		o += " BLK_TC_QUEUE"
	}
	if in2&(1<<5) > 0 {
		o += " BLK_TC_REQUEUE"
	}
	if in2&(1<<6) > 0 {
		o += " BLK_TC_ISSUE"
	}
	if in2&(1<<7) > 0 {
		o += " BLK_TC_COMPLETE"
	}
	if in2&(1<<8) > 0 {
		o += " BLK_TC_FS"
	}
	if in2&(1<<9) > 0 {
		o += " BLK_TC_PC"
	}
	if in2&(1<<10) > 0 {
		o += " BLK_TC_NOTIFY"
	}
	if in2&(1<<11) > 0 {
		o += " BLK_TC_AHEAD"
	}
	if in2&(1<<12) > 0 {
		o += " BLK_TC_META"
	}
	if in2&(1<<13) > 0 {
		o += " BLK_TC_DISCARD"
	}
	if in2&(1<<14) > 0 {
		o += " BLK_TC_DRV_DATA"
	}
	if in2&(1<<15) > 0 {
		o += " BLK_TC_END"
	}
	if in2&(1<<16) > 0 {
		o += " BLK_TA_QUEUE"
	}
	if in2&(1<<17) > 0 {
		o += " BLK_TA_BACKMERGE"
	}
	if in2&(1<<18) > 0 {
		o += " BLK_TA_FRONTMERGE"
	}
	if in2&(1<<19) > 0 {
		o += " BLK_TA_GETRQ"
	}
	if in2&(1<<20) > 0 {
		o += " BLK_TA_SLEEPRQ"
	}
	if in2&(1<<21) > 0 {
		o += " BLK_TA_REQUEUE"
	}
	if in2&(1<<22) > 0 {
		o += " BLK_TA_ISSUE"
	}
	if in2&(1<<23) > 0 {
		o += " BLK_TA_COMPLETE"
	}
	if in2&(1<<24) > 0 {
		o += " BLK_TA_PLUG"
	}
	if in2&(1<<25) > 0 {
		o += " BLK_TA_UNPLUG_IO"
	}
	if in2&(1<<26) > 0 {
		o += " BLK_TA_UNPLUG_TIMER"
	}
	if in2&(1<<27) > 0 {
		o += " BLK_TA_INSERT"
	}
	if in2&(1<<28) > 0 {
		o += " BLK_TA_SPLIT"
	}
	if in2&(1<<29) > 0 {
		o += " BLK_TA_BOUNCE"
	}
	if in2&(1<<30) > 0 {
		o += " BLK_TA_REMAP"
	}
	if in2&(1<<31) > 0 {
		o += " BLK_TA_ABORT"
	}

	return o
}

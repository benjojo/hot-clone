package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

var (
	reassemblePath   = flag.String("reassemble", "", "use this hot-clone backup file to restore into a file or block device")
	reassembleOutput = flag.String("reassemble-output", "", "The path of the file or block device that is going to be restored to")
)

func reassembleMain() {
	imageFd, err := os.Open(*reassemblePath)
	if err != nil {
		log.Fatalf("Can't open image file -reassemble %v -- %v", *reassembleOutput, err)
	}

	if *reassembleOutput == "" {
		log.Fatalf("You must provide a -reassemble-output to restore to")
	}

	// First we should 100% check that we are dealing with a hot-clone image file
	ReadBanner := ""
	for {
		b := make([]byte, 1)
		n, err := imageFd.Read(b)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Failed to read image banner %v", err)
		}
		if n == 1 {
			ReadBanner += string(b)
		}

		if b[0] == '\n' {
			break
		}
	}

	if !strings.Contains(ReadBanner, "Hot-Clone") {
		log.Fatalf("This image does not seem to be the output of hot-clone")
	}

	outputStat, err := os.Stat(*reassembleOutput)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Fatalf("Unable to stat output target %v", err)
		}
	}

	var outputFD *os.File
	if outputStat != nil {
		if outputStat.Mode()&os.ModeDevice != 0 {
			outputFD, err = os.OpenFile(*reassembleOutput, os.O_RDWR, 0777)
		} else {
			outputFD, err = os.Create(*reassembleOutput)
		}
	} else {
		outputFD, err = os.Create(*reassembleOutput)
	}

	if err != nil {
		log.Fatalf("Can't open/create output %v", err)
	}

	n := 0
	for {
		ReadHeader := ""
		ReadEOF := false
		for {
			b := make([]byte, 1)
			n, err := imageFd.Read(b)
			if err != nil {
				if err == io.EOF {
					ReadEOF = true
					break
				}
				log.Fatalf("Failed to read image header %v", err)
			}
			if n == 1 {
				ReadHeader += string(b)
			}

			if b[0] == '\n' {
				break
			}
		}

		if ReadEOF {
			break
		}

		SectorStart := 0
		BytesLeftToRead := 0
		parsed, err := fmt.Sscanf(ReadHeader, "S:%d\tL:%d\n", &SectorStart, &BytesLeftToRead)
		if parsed != 2 {
			log.Fatalf("Failed to parse header (%v) -- aborting (%v - %v - %v)", ReadHeader, err, SectorStart, BytesLeftToRead)
		}

		_, err = outputFD.Seek(int64(SectorStart*512), 0)
		if err != nil {
			log.Fatalf("Seek failure (to %d) on output file/device %v", SectorStart, err)
		}
		if n == 0 || n%1000 == 0 {
			if *debug {
				log.Printf("Restoring section (Sector: %v (len %d bytes) (debug: '%s')", SectorStart, BytesLeftToRead, strings.Trim(ReadHeader, "\n"))
			} else {
				log.Printf("Restoring section (Sector: %v (len %d bytes)", SectorStart, BytesLeftToRead)
			}
		}
		n++

		var buf []byte
		for {
			expectedRead := 4096
			if BytesLeftToRead < 4096 {
				expectedRead = BytesLeftToRead
			}

			if BytesLeftToRead == 0 {
				break
			}

			buf = make([]byte, expectedRead)

			n, err := imageFd.Read(buf)
			if err != nil {
				log.Fatalf("Image read failure -- %v", err)
			}
			if n != expectedRead {
				log.Fatalf("Image short read failure -- %v != %v (had %d bytes left)", n, expectedRead, BytesLeftToRead)
			}
			BytesLeftToRead = BytesLeftToRead - expectedRead

			_, err = outputFD.Write(buf)
			if err != nil {
				log.Fatalf("Output file/device write failure -- %v", err)
			}
		}

	}

}

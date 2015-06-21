package main

import (
	"syscall"

	"errors"
	"flag"
	"os"
	"os/signal"

	"hash/crc32"
	"io"
	"log"
)

var (
	path = flag.String("path", "test", "path to fifo")
)

func readFifo() (err error) {
	if err := syscall.Mkfifo(*path, 0666); err != nil {
		return err
	}

	// remove the fifo when finished
	defer os.Remove(*path)

	// listen for user interrupts
	c := make(chan os.Signal, 1)
	errorChannel := make(chan error)
	signal.Notify(c, os.Interrupt)

	go func() {
		<-c
		errorChannel <- errors.New("Interrupt")
	}()

	// try to open queue
	var f *os.File

	go func(d chan error) {
		var err error
		f, err = os.Open(*path)
		if err != nil {
			d <- err
		}
		d <- nil
	}(errorChannel)

	log.Println("listening...")

	// wait for opening of fifo
	// if we get interrupted, we will get notified here
	if err = <-errorChannel; err != nil {
		return err
	}

	// schedule file closing
	defer f.Close()

	// try to read from file till EOF
	var (
		buffer                        [1024]byte
		duplicates, lastHash, hash, n        = 0, 0, 0, 0
		offset, start                 uint64 = 0, 0
	)

	for n, err = f.Read(buffer[:]); n != 0 && err == nil; {
		// hash the blocks of data we get
		hash = int(crc32.ChecksumIEEE(buffer[:]))
		if hash == lastHash || lastHash == 0 {
			if duplicates == 0 {
				start = offset
			}

			duplicates++
		} else {
			if duplicates > 1 {
				log.Println("found", duplicates, "in a row (from", start, "to", offset, ")")
			}

			duplicates = 0
			start = offset
		}
	
		lastHash = hash
		offset += uint64(n)

		// look for user interrupt
		select {
		case err = <-errorChannel:
		default:
			n, err = f.Read(buffer[:])
		}
	}

	if (duplicates > 0) {
		log.Println("found", duplicates, "in a row (from", start, "to", offset, ")")
	}
	// if we encountered EOF, we ignore the error
	// as that is to be expected
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

func main() {
	flag.Parse()
	log.Println("creating fifo at", *path)

	var err error
	
	for ;err == nil; err = readFifo() {}
	
	log.Fatal(err);
}

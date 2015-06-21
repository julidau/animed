package main

import (
	"io"
	"log"
)

type Downloader interface {
	Download(in io.Reader, out io.Writer) error
}

type DCCDownloader struct {
	start, size, current uint64
}

func (loader *DCCDownloader) Download(in io.Reader, out io.Writer) error {
	var (
		number [2]int
		err    error = nil

		buffer []byte = make([]byte, 1024)
	)

	log.Println(loader.current)

	for loader.current < loader.size {
		number[0], err = in.Read(buffer)

		if number[0] == 0 || err != nil {
			break
		}

		// write read data back to output
		for number[1] = 0; number[1] < number[0] && err == nil; {
			number[1], err = out.Write(buffer[:number[0]])
		}

		if number[0] != number[1] || err != nil {
			break
		}

		loader.current += uint64(number[0])
	}

	log.Println(loader.current, loader.size)
	log.Println(err)
	log.Println(number)

	if err == io.EOF {
		return nil
	}

	return err
}

package main

import (
	"bufio"
	"encoding/csv"
	"os"
)

var (
	writeChan chan []string
)

func initWriter() (chan<- []string, chan interface{}) {
	name := *csvFile
	c := make(chan []string, 10000)
	doneChan := make(chan interface{})

	writeChan = c

	go func() {
		f, err := os.Create(name)
		if err != nil {
			panic("Could not open file for writing the CSV output")
		}
		defer f.Close()

		bf := bufio.NewWriter(f)
		defer bf.Flush()

		writer := csv.NewWriter(bf)
		for fields := range c {
			writer.Write(fields)

		}

		doneChan <- nil
	}()

	return c, doneChan
}

func write(record []string) {
	writeChan <- record
}

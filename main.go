package main

import (
	"bufio"
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"golang.org/x/sync/errgroup"
)

const CITT = 0xFFFF

func init() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: citt [-h] <file...>")
		os.Exit(2)
	}
}

func main() {
	parallel := flag.Int("parallel", 10, "parallel task")
	flag.Parse()

	if *parallel <= 0 {
		*parallel = 1
	}
	var (
		sema = make(chan struct{}, *parallel)
		grp  errgroup.Group
	)
	for _, a := range flag.Args() {
		sema <- struct{}{}
		a := a
		grp.Go(func() error {
			citt, sum, err := Calculate(a)
			<-sema
			if err != nil {
				return err
			}
			fmt.Printf("%s: crc: %#x (%[2]d) - md5: %x\n", a, citt, sum)
			return nil
		})
	}
	if err := grp.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func Calculate(p string) (uint16, []byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return CITT, nil, err
	}
	defer f.Close()

	var (
		w = md5.New()
		r = io.TeeReader(f, w)
	)
	v, err := calculate(r)
	if err != nil && !errors.Is(err, io.EOF) {
		return v, nil, err
	}
	s := w.Sum(nil)

	return v, s[:], nil
}

func calculate(r io.Reader) (uint16, error) {
	rs := bufio.NewReader(r)

	v := uint16(CITT)
	for {
		b, err := rs.ReadByte()
		switch err {
		case nil:
			x := (v >> 8) ^ uint16(b)
			x ^= x >> 4
			v = (v << 8) ^ (x << 12) ^ (x << 5) ^ x
		case io.EOF:
			return v, nil
		default:
			return v, err
		}
	}
}

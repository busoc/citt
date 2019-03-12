package main

import (
	"bufio"
	"crypto/md5"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sync/errgroup"
)

const CITT = 0xFFFF

var (
	Version   string = ""
	BuildTime string = ""
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stderr)

	version := flag.Bool("version", false, "print version and exit")
	parallel := flag.Int("parallel", 10, "parallel task")
	flag.Usage = func() {
		log.Printf("usage: %s [-version] <file...>", filepath.Base(os.Args[0]))
		os.Exit(1)
	}
	flag.Parse()
	if *version {
		log.Printf("%s version %s (%s)", filepath.Base(os.Args[0]), Version, BuildTime)
		os.Exit(1)
	}
	if flag.NArg() == 0 {
		flag.Usage()
	}
	var g errgroup.Group

	if *parallel <= 0 {
		*parallel = 1
	}
	sema := make(chan struct{}, *parallel)
	for _, a := range flag.Args() {
		a := a
		sema <- struct{}{}
		g.Go(func() error {
			s, bs, err := Calculate(a)
			<-sema
			if err != nil {
				log.Fatalln(err)
			}
			if err != nil {
				return err
			}
			log.Printf("%s: crc: %#x (%[2]d) - md5: %x", a, s, bs)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		log.Fatalln(err)
	}
}

func Calculate(p string) (uint16, []byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return CITT, nil, err
	}
	defer f.Close()
	w := md5.New()

	r := io.TeeReader(f, w)
	v, err := calculate(r)
	if err != nil && err != io.EOF {
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

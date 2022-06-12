package main

import (
	"fmt"
	"log"
	"os"
)

type driver struct {
	*log.Logger
}

func newDriver() *driver {
	a := &driver{
		log.New(os.Stdout, "Got Command Error", log.Ldate|log.Ltime|log.Lmsgprefix),
	}
	return a
}

func (d *driver) Drive() error {
	if len(os.Args) < 2 {
		str := `odor expects two or three arguments: 
				1: the path to the torrent file, 
				2: the path where you wuld have the downloaded file(s) saved (optional)`
		d.Printf("%s\n", str)
		return fmt.Errorf(str)
	}
	var torrPath, fPath string
	torrPath = os.Args[1]
	if len(os.Args) == 3 {
		fPath = os.Args[2]
	} else {
		path, err := os.Getwd()
		fPath = path
		if err != nil {
			s := "could not get working directory"
			d.Println(s)
			return fmt.Errorf(s)
		}
	}

	t, err := NewTorrent(torrPath, fPath)
	if err != nil {
		d.Printf("%s\n", err.Error())
		return err
	}

	if err := t.Start(); err != nil {
		d.Printf("%s\n", err.Error())
		return err
	}

	return nil
}

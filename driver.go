package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
		path, exists := os.LookupEnv("HOME")
		if !exists {
			s := "could not get home directory"
			d.Println(s)
			return fmt.Errorf(s)
		}
		path = filepath.Join(path, "Odor")
		if err := os.MkdirAll(path, 0700); err != nil {
			return err
		}

	}
	ctx := context.TODO()
	t, err := NewTorrent(ctx, torrPath, fPath)
	if err != nil {
		d.Printf("%s\n", err.Error())
		return err
	}
	d.Println("Torrent download begins...")
	if err := t.Start(ctx); err != nil {
		d.Printf("%s\n", err.Error())
		return err
	}
	d.Printf("Torrent %s save in directory: %s", t.name, t.fPath)
	return nil
}

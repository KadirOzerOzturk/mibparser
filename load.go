package mibparser

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

func Load(opts ...Option) (*MIBParser, error) {
	opt := Opts{}
	for _, o := range opts {
		o(&opt)
	}
	return &MIBParser{opts: opt}, nil
}

func (p *MIBParser) ReadMIBFile() ([]string, error) {
	files, err := ioutil.ReadDir(p.opts.Path)
	if err != nil {
		log.Fatal(err)
	}
	var mergedMIB []string
	for _, file := range files {
		lines, err := readMIBFileWithPath(p.opts.Path + "/" + file.Name())
		if err != nil {
			fmt.Println("Error reading MIB file:", err)
		}
		mergedMIB = append(mergedMIB, lines...)
	}
	return mergedMIB, nil
}

func readMIBFileWithPath(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

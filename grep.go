package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
)

var Log *log.Logger = log.New(os.Stdout, `DBG: `, 3)

type Find struct {
	Out chan string
}

type Grep struct {
	In   chan string
	Done chan struct{}
}

func main() {
	grepChan := make(chan string, 5000)
	done := make(chan struct{})

	go Find{
		Out: grepChan,
	}.Start(".", regexp.MustCompile(`.*\.java$`))

	go Grep{
		In:   grepChan,
		Done: done,
	}.Start(regexp.MustCompile(`android`))

	<-done
	Log.Println("END")
}

func (f Find) Start(root string, pattern *regexp.Regexp) {
	Log.Printf("Find Start() : %s", pattern)
	defer close(f.Out)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if !pattern.MatchString(path) {
			return nil
		}
		Log.Printf("Match %s\n", path)
		f.Out <- path
		return nil
	})
	Log.Println("Find Start() END")
}

func (g Grep) Start(pattern *regexp.Regexp) {
	Log.Printf("Grep Start() : %s", pattern)
	sem := make(chan struct{}, 208)
	wg := &sync.WaitGroup{}
	for path := range g.In {
		sem <- struct{}{}
		wg.Add(1)
		go g.Grep(path, pattern, sem, wg)
	}
	wg.Wait()
	g.Done <- struct{}{}
	Log.Println("Grep Start() END")
}

func (g Grep) Grep(path string, pattern *regexp.Regexp, sem chan struct{}, wg *sync.WaitGroup) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	defer func() {
		f.Close()
		<-sem
		wg.Done()
	}()

	buf := bufio.NewReader(f)
	for {
		l, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		if pattern.Match(l) {
			fmt.Println(string(l))
		}
	}
	return
}

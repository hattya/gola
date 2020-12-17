//
// gola.go - A script launcher written in Go
//
//   Copyright (c) 2011-2020 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		return
	}
	g := newGola(os.Args[0], os.Args[1])
	os.Exit(g.exec(os.Args[1:]))
}

type gola struct {
	name   string
	ext    string
	config struct {
		Dir []string
		Map map[string]map[string]string
	}
}

func newGola(argv0, name string) *gola {
	g := &gola{
		name: name,
		ext:  filepath.Ext(name),
	}
	g.loadConfig(argv0)
	// redirect to a found file
	if !g.isFile(g.name) {
		ok := false
		for _, n := range g.config.Dir {
			name := filepath.Join(g.name, n)
			if g.isFile(name) {
				g.name = name
				ok = true
				break
			}
		}
		if !ok {
			log.Fatalf("'%v' is not a file", g.name)
		}
	}
	return g
}

func (g *gola) loadConfig(argv0 string) {
	if !filepath.IsAbs(argv0) {
		var abs string
		var err error
		if g.isExe(argv0) {
			abs, err = filepath.Abs(argv0)
		} else {
			abs, err = exec.LookPath(argv0)
		}
		if err != nil {
			log.Fatal(err)
		}
		argv0 = abs
	}
	name := argv0[:len(argv0)-len(filepath.Ext(argv0))] + ".json"
	if !g.isFile(name) {
		var home string
		switch runtime.GOOS {
		case "windows":
			home = os.Getenv("APPDATA")
		default:
			home = os.Getenv("XDG_CONFIG_HOME")
			if home == "" {
				home = filepath.Join(os.Getenv("HOME"), ".config")
			}
		}
		name = filepath.Join(home, "gola", "settings.json")
	}
	// read config
	if !g.isFile(name) {
		return
	}
	buf, err := ioutil.ReadFile(name)
	if err != nil {
		log.Fatalf("could not read '%v'", name)
	}
	// parse config
	err = json.Unmarshal(buf, &g.config)
	if err != nil {
		log.Fatalf("could not unmarshal '%v': %v", name, err)
	}
}

func (g *gola) isExe(name string) bool {
	switch {
	case g.isFile(name):
		return true
	case runtime.GOOS == "windows":
		return g.isFile(name + ".exe")
	}
	return false
}

func (g *gola) isFile(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && !fi.IsDir()
}

func (g *gola) exec(args []string) int {
	kwd, argv, err := g.loadScript()
	switch {
	case err != nil:
		log.Fatal(err)
	case len(argv) == 0:
		log.Fatalf("could not find interpreter[%v] for '%v'", kwd, g.name)
	}
	argv = append(argv, args...)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if !e.Exited() {
				log.Fatal(err)
			}
			return e.Sys().(syscall.WaitStatus).ExitStatus()
		}
		log.Fatal(err)
	}
	return 0
}

func (g *gola) loadScript() (kwd string, argv []string, err error) {
	argv, err = g.parseShebang()
	if err != nil || len(argv) == 0 {
		return
	}
	kwd = filepath.Base(argv[0])
	i := 0
	// skip env
	if kwd == "env" || kwd == "env.exe" {
		i++
		// skip args
		for i < len(argv) && strings.HasPrefix(argv[i], "-") {
			if argv[i] == "-u" {
				i += 2
			} else {
				i++
			}
		}
		// skip NAME=VALUE
		for i < len(argv) && strings.Contains(argv[i], "=") {
			i++
		}
		kwd = filepath.Base(argv[i])
	}
	// find interpreter from map
	for {
		if _, ok := g.config.Map[kwd]; ok {
			break
		}
		ext := filepath.Ext(kwd)
		if ext == "" {
			break
		}
		kwd = kwd[:len(kwd)-len(ext)]
	}
	if name, ok := g.config.Map[kwd][g.ext]; ok {
		argv[i] = name
	} else if name, ok := g.config.Map[kwd][""]; ok {
		argv[i] = name
	} else {
		argv = nil
		return
	}
	argv = argv[i:]
	return
}

func (g *gola) parseShebang() (argv []string, err error) {
	// read shebang
	shebang, err := g.readShebang()
	if err != nil || !strings.HasPrefix(shebang, "#!") {
		return
	}
	// parse shebang
	for _, s := range strings.Fields(strings.Replace(shebang[2:], "\\", "/", -1)) {
		if len(argv) == 1 && filepath.IsAbs(argv[0]) && strings.Contains(s, "/") {
			// join a path which contains spaces
			argv[0] += " " + s
		} else {
			argv = append(argv, s)
		}
	}
	return
}

func (g *gola) readShebang() (shebang string, err error) {
	f, err := os.Open(g.name)
	if err != nil {
		return
	}
	defer f.Close()
	// check signature
	b := make([]byte, 2)
	if _, err = f.ReadAt(b, 0); err != nil {
		err = fmt.Errorf("exec format error: '%v'", g.name)
		return
	}
	var br *bufio.Reader
	switch string(b) {
	case "#!":
		br = bufio.NewReader(f)
	case "PK":
		var size int64
		size, err = f.Seek(0, io.SeekEnd)
		if err != nil {
			return
		}
		var zr *zip.Reader
		zr, err = zip.NewReader(f, size)
		if err != nil {
			return
		}
		var zf *zip.File
		for _, zf = range zr.File {
			for _, n := range g.config.Dir {
				if zf.Name == n {
					goto Found
				}
			}
		}
		return
	Found:
		var rc io.ReadCloser
		rc, err = zf.Open()
		if err != nil {
			err = fmt.Errorf("could not open '%v' in '%v': %v", zf.Name, g.name, err)
			return
		}
		defer rc.Close()
		br = bufio.NewReader(rc)
	default:
		return
	}
	// read fist line
	if shebang, err = br.ReadString('\n'); err == io.EOF {
		err = nil
	}
	return
}

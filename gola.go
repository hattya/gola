//
// gola.go - A script launcher written in Go
//
//   Copyright (c) 2011-2014 Akinori Hattori <hattya@gmail.com>
//
//   Permission is hereby granted, free of charge, to any person
//   obtaining a copy of this software and associated documentation files
//   (the "Software"), to deal in the Software without restriction,
//   including without limitation the rights to use, copy, modify, merge,
//   publish, distribute, sublicense, and/or sell copies of the Software,
//   and to permit persons to whom the Software is furnished to do so,
//   subject to the following conditions:
//
//   The above copyright notice and this permission notice shall be
//   included in all copies or substantial portions of the Software.
//
//   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
//   EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
//   MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
//   NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
//   BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
//   ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
//   CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
//   SOFTWARE.
//

package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
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
	gola := newGola(os.Args[0], os.Args[1])
	os.Exit(gola.exec(os.Args[1:]))
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
	kwd, argv := g.loadScript()
	if len(argv) == 0 {
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

func (g *gola) loadScript() (kwd string, argv []string) {
	argv = g.parseShebang()
	if len(argv) == 0 {
		return
	}
	kwd = filepath.Base(argv[0])
	i := 0
	// skip env
	if kwd == "env" || kwd == "env.exe" {
		// skip args
		for i++; i < len(argv) && strings.HasPrefix(argv[i], "-"); i++ {
		}
		if 0 < i && argv[i-1] == "-" {
			// skip NAME=VALUE
			for ; i < len(argv) && strings.Contains(argv[i], "="); i++ {
			}
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
	if _, ok := g.config.Map[kwd]; !ok {
		return kwd, nil
	} else if name, ok := g.config.Map[kwd][g.ext]; ok {
		argv[i] = name
	} else if name, ok := g.config.Map[kwd][""]; ok {
		argv[i] = name
	} else {
		return kwd, nil
	}
	return kwd, argv[i:]
}

func (g *gola) parseShebang() (argv []string) {
	// read shebang
	shebang := g.readShebang()
	if shebang == "" || !strings.HasPrefix(shebang, "#!") {
		return
	}
	shebang = strings.Replace(shebang, "\\", "/", -1)
	// parse shebang
	for _, s := range strings.Fields(shebang[2:]) {
		if len(argv) == 1 && filepath.IsAbs(argv[0]) && strings.Contains(s, "/") {
			// join a path which contains spaces
			argv[0] += " " + s
		} else {
			argv = append(argv, s)
		}
	}
	return argv
}

func (g *gola) readShebang() (shebang string) {
	f, err := os.Open(g.name)
	if err != nil {
		log.Fatalf("could not open '%v'", g.name)
	}
	defer f.Close()
	// check signature
	b := make([]byte, 2)
	switch n, err := f.ReadAt(b, 0); {
	case n != len(b):
		log.Fatalf("exec format error: '%v'", g.name)
	case err != nil:
		log.Fatal(err)
	}
	var br *bufio.Reader
	switch string(b) {
	case "#!":
		br = bufio.NewReader(f)
	case "PK":
		size, err := f.Seek(0, 2)
		if err != nil {
			log.Fatal(err)
		}
		zr, err := zip.NewReader(f, size)
		if err != nil {
			log.Fatal(err)
		}
		zf := func() *zip.File {
			for _, zf := range zr.File {
				for _, n := range g.config.Dir {
					if zf.Name == n {
						return zf
					}
				}
			}
			return nil
		}()
		if zf == nil {
			return
		}
		rc, err := zf.Open()
		if err != nil {
			log.Fatalf("could not open '%v' in '%v'", zf.Name, g.name)
		}
		defer rc.Close()
		br = bufio.NewReader(rc)
	default:
		return
	}
	// read fist line
	line, err := br.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	return line
}

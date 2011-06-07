//
// gola.go - A script launcher written in Go
//
//   Copyright (c) 2011 Akinori Hattori <hattya@gmail.com>
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
	"exec"
	"io/ioutil"
	"json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		return
	}
	gola := newGola(os.Args[0], os.Args[1])
	rv := gola.exec(os.Args[1:])
	os.Exit(rv)
}

type gola struct {
	dir  []string
	map_ map[string]map[string]string
	name string
	ext  string
}

func newGola(argv0, name string) *gola {
	g := new(gola)
	g.name = name
	g.ext = filepath.Ext(name)
	g.loadConfig(argv0)
	// redirect to a found file
	if !g.isFile(g.name) {
		name := func() string {
			for _, n := range g.dir {
				name := filepath.Join(g.name, n)
				if g.isFile(name) {
					return name
				}
			}
			return ""
		}()
		if name == "" {
			log.Fatalf("'%v' is not a file", g.name)
		}
		g.name = name
	}
	return g
}

func (g *gola) isFile(name string) bool {
	fi, err := os.Stat(name)
	return err == nil && fi.IsRegular()
}

func (g *gola) loadConfig(argv0 string) {
	if !filepath.IsAbs(argv0) {
		if g.isExe(argv0) {
			abs, err := filepath.Abs(filepath.Clean(argv0))
			if err != nil {
				log.Fatal("could not get current directory")
			}
			argv0 = abs
		} else {
			abs, err := exec.LookPath(argv0)
			if err != nil {
				log.Fatalf("could not find '%v' in $PATH", argv0)
			}
			argv0 = abs
		}
	}
	name := argv0[:len(argv0)-len(filepath.Ext(argv0))] + ".json"
	if !g.isFile(name) {
		var home string
		if runtime.GOOS == "windows" {
			home = os.Getenv("APPDATA")
		} else {
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
	v := map[string]interface{}{}
	err = json.Unmarshal(buf, &v)
	if err != nil {
		log.Fatalf("could not unmarshal '%v': %v", name, err)
	}
	// section: dir
	ok := func() bool {
		dir, ok := v["dir"].([]interface{})
		if !ok {
			return false
		}
		g.dir = make([]string, len(dir))
		for i, v := range dir {
			x, ok := v.(string)
			if !ok {
				return false
			}
			g.dir[i] = x
		}
		return true
	}()
	if !ok {
		log.Fatal("'dir' option must be a string array")
	}
	// section: map
	ok = func() bool {
		map_, ok := v["map"].(map[string]interface{})
		if !ok {
			return false
		}
		g.map_ = map[string]map[string]string{}
		for kw, obj := range map_ {
			g.map_[kw] = map[string]string{}
			nobj, ok := obj.(map[string]interface{})
			if !ok {
				return false
			}
			for k, v := range nobj {
				x, ok := v.(string)
				if !ok {
					return false
				}
				g.map_[kw][k] = x
			}
		}
		return true
	}()
	if !ok {
		log.Fatal("'map' option should be a nested string value object")
	}
}

func (g *gola) isExe(name string) bool {
	if g.isFile(name) {
		return true
	} else if runtime.GOOS == "windows" {
		return g.isFile(name + ".exe")
	}
	return false
}

func (g *gola) exec(args []string) int {
	kwd, argv := g.loadScript()
	if len(argv) == 0 {
		log.Fatalf("could not find interpreter[%v] for '%v'", kwd, g.name)
	}
	for _, v := range args {
		argv = append(argv, v)
	}
	cmd, err := exec.Run(argv[0], argv, os.Environ(), "", exec.PassThrough,
		exec.PassThrough, exec.PassThrough)
	if err != nil {
		log.Fatal(err)
	}
	waitmsg, err := cmd.Wait(0)
	if err != nil {
		log.Fatal(err)
	}
	return waitmsg.ExitStatus()
}

func (g *gola) loadScript() (kwd string, argv []string) {
	argv = g.parseShebang()
	if len(argv) == 0 {
		return "", []string{}
	}
	i := 0
	kwd = filepath.Base(argv[i])
	// skip env
	if kwd == "env" || kwd == "env.exe" {
		// skip args
		for i++; i < len(argv) && strings.HasPrefix(argv[i], "-"); i++ {
		}
		if i > 0 && argv[i-1] == "-" {
			// skip NAME=VALUE
			for ; i < len(argv) && strings.Contains(argv[i], "="); i++ {
			}
		}
		kwd = filepath.Base(argv[i])
	}
	// find interpreter from map
	for {
		if _, ok := g.map_[kwd]; ok {
			break
		}
		x := filepath.Ext(kwd)
		if x == "" {
			break
		}
		kwd = kwd[:len(kwd)-len(x)]
	}
	if _, ok := g.map_[kwd]; !ok {
		return kwd, []string{}
	} else if name, ok := g.map_[kwd][g.ext]; ok {
		argv[i] = name
	} else if name, ok := g.map_[kwd][""]; ok {
		argv[i] = name
	} else {
		return kwd, []string{}
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
	p := strings.Split(strings.TrimSpace(shebang[2:]), " ", -1)
	for _, s := range p {
		if len(argv) == 1 &&
			filepath.IsAbs(argv[0]) &&
			!filepath.IsAbs(s) &&
			strings.Contains(s, "/") {
			// join a path which contains spaces
			argv[0] += " " + s
		} else {
			argv = append(argv, s)
		}
	}
	return argv
}

func (g *gola) readShebang() (shebang string) {
	file, err := os.Open(g.name)
	if err != nil {
		log.Fatalf("could not open '%v'", g.name)
	}
	defer file.Close()
	// check signature
	b := make([]byte, 2)
	n, err := file.ReadAt(b, 0)
	if n != len(b) {
		log.Fatalf("could not read %v bytes from '%v'", len(b), g.name)
	} else if err != nil {
		log.Fatalf("could not read from '%v': %v", g.name, err)
	}
	var br *bufio.Reader
	if string(b) == "#!" {
		// read fist line
		br = bufio.NewReader(file)
	} else if string(b) == "PK" {
		fi, err := file.Stat()
		if err != nil {
			log.Fatalf("could not access '%v': %v", g.name, err)
		}
		zr, err := zip.NewReader(file, fi.Size)
		if err != nil {
			log.Fatalf("could not open zip file '%v': %v", g.name, err)
		}
		zf := func() *zip.File {
			for _, file := range zr.File {
				for _, n := range g.dir {
					if n == file.Name {
						return file
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
	} else {
		return
	}
	bytes, isPrefix, err := br.ReadLine()
	if isPrefix {
		log.Fatalf("too long line: '%v'", g.name)
	} else if err != nil {
		log.Fatalf("could not read from '%v': %v", g.name, err)
	}
	return string(bytes)
}

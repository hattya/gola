//
// gola :: gola_test.go
//
//   Copyright (c) 2020 Akinori Hattori <hattya@gmail.com>
//
//   SPDX-License-Identifier: MIT
//

package main

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestNewGola(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	argv0 := filepath.Join(dir, "gola")
	json := filepath.Join(dir, "gola.json")
	exe := argv0
	switch runtime.GOOS {
	case "windows":
		exe += ".exe"
	}
	if err := ioutil.WriteFile(exe, []byte{}, 0777); err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(dir, "a")
	if err := os.Mkdir(name, 0777); err != nil {
		t.Fatal(err)
	}
	if err := file(filepath.Join(name, "__main__.py"), "#!/usr/bin/env python\n"); err != nil {
		t.Fatal(err)
	}
	// invalid config
	if err := file(json, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := newGola(argv0, name); err == nil {
		t.Error("expected error")
	}
	// not found
	if err := file(json, `{}`); err != nil {
		t.Fatal(err)
	}
	if _, err := newGola(argv0, name); err == nil {
		t.Error("expected error")
	}
	// redirect
	if err := file(json, `{"dir": ["__main__.py"]}`); err != nil {
		t.Fatal(err)
	}
	if g, err := newGola(argv0, name); err != nil {
		t.Error(err)
	} else if g, e := g.name, filepath.Join(name, "__main__.py"); g != e {
		t.Errorf("expected %q, got %q", e, g)
	}
}

func TestLoadConfig(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	path := os.Getenv("PATH")
	defer os.Setenv("PATH", path)
	os.Setenv("PATH", dir+string(os.PathListSeparator)+path)

	var argv0 string
	switch runtime.GOOS {
	case "windows":
		argv0 = filepath.Join(dir, "gola.exe")
	default:
		argv0 = filepath.Join(dir, "gola")
	}
	json := filepath.Join(dir, "gola.json")

	g := &gola{}
	if err := g.loadConfig(argv0); err != nil {
		t.Error(err)
	}
	if err := g.loadConfig(filepath.Base(argv0)); err == nil {
		t.Error("expected error")
	}
	// invalid config
	if err := ioutil.WriteFile(argv0, []byte{}, 0777); err != nil {
		t.Fatal(err)
	}
	if err := file(json, ""); err != nil {
		t.Fatal(err)
	}
	// :: abs
	if err := g.loadConfig(argv0); err == nil {
		t.Error("expected error")
	}
	// :: rel
	popd, err := pushd(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.loadConfig(filepath.Base(argv0)); err == nil {
		t.Error("expected error")
	}
	if err := popd(); err != nil {
		t.Fatal(err)
	}
	// :: lookup PATH
	if err := g.loadConfig(filepath.Base(argv0)); err == nil {
		t.Error("expected error")
	}
	// valid config
	if err := file(json, "{}"); err != nil {
		t.Fatal(err)
	}
	// :: abs
	if err := g.loadConfig(argv0); err != nil {
		t.Error(err)
	}
	// :: rel
	popd, err = pushd(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.loadConfig(filepath.Base(argv0)); err != nil {
		t.Error(err)
	}
	if err := popd(); err != nil {
		t.Fatal(err)
	}
	// :: lookup PATH
	if err := g.loadConfig(filepath.Base(argv0)); err != nil {
		t.Error(err)
	}
}

func TestExec(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	stdout := os.Stdout
	defer func() { os.Stdout = stdout }()
	os.Stdout = nil

	g := &gola{}
	if err := g.exec([]string{}); err == nil {
		t.Error("expected error")
	}

	g.name = filepath.Join(dir, "a.go")
	g.ext = filepath.Ext(g.name)
	if err := file(g.name, "#!/usr/bin/env go\n"); err != nil {
		t.Fatal(err)
	}
	if err := g.exec([]string{}); err == nil {
		t.Error("expected error")
	}

	g.config.Map = map[string]map[string]string{
		"go": {
			"": "go",
		},
	}
	if err := g.exec([]string{"version"}); err != nil {
		t.Error(err)
	}
}

func TestLoadScript(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	g := &gola{}
	if _, _, err := g.loadScript(); err == nil {
		t.Error("expected error")
	}

	g.name = filepath.Join(dir, "a.py")
	g.ext = filepath.Ext(g.name)
	for _, data := range []string{
		"#!/usr/bin/env -u FOO BAR=BAZ python\n",
		"#!/usr/bin/env -i BAR=BAZ python\n",
	} {
		if err := file(g.name, data); err != nil {
			t.Fatal(err)
		}
		if kwd, argv, err := g.loadScript(); err != nil {
			t.Error(err)
		} else if g, e := kwd, "python"; g != e {
			t.Errorf("expected %q, got %q", e, g)
		} else if g, e := argv, []string(nil); !reflect.DeepEqual(g, e) {
			t.Errorf("expected %#v, got %#v", e, g)
		}
	}

	if err := file(g.name, "#!/usr/bin/env python3.9"); err != nil {
		t.Fatal(err)
	}
	g.config.Map = map[string]map[string]string{
		"python3": {
			".py":  `C:\Program Files\Python\python.exe`,
			".pyw": `C:\Program Files\Python\pythonw.exe`,
			"":     `C:\Windows\py.exe`,
		},
	}
	for _, tt := range []struct {
		ext  string
		argv []string
	}{
		{".py", []string{g.config.Map["python3"][".py"]}},
		{".pyz", []string{g.config.Map["python3"][""]}},
		{".pyw", []string{g.config.Map["python3"][".pyw"]}},
		{".pyzw", []string{g.config.Map["python3"][""]}},
	} {
		g.ext = tt.ext
		if kwd, argv, err := g.loadScript(); err != nil {
			t.Error(err)
		} else if g, e := kwd, "python3"; g != e {
			t.Errorf("expected %q, got %q", e, g)
		} else if !reflect.DeepEqual(argv, tt.argv) {
			t.Errorf("expected %#v, got %#v", tt.argv, argv)
		}
	}
}

func TestParseShebang(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	g := &gola{}
	if _, err := g.parseShebang(); err == nil {
		t.Error("expected error")
	}

	var C string
	switch runtime.GOOS {
	case "windows":
		C = "C:"
	default:
		C = "/c"
	}
	g.name = filepath.Join(dir, "a.py")
	for _, tt := range []struct {
		shebang string
		argv    []string
	}{
		{"#!/usr/bin/env python", []string{"/usr/bin/env", "python"}},
		{"#!" + filepath.FromSlash(C+"/Program Files/Python/python.exe"), []string{C + "/Program Files/Python/python.exe"}},
	} {
		if err := file(g.name, tt.shebang+"\n"); err != nil {
			t.Fatal(err)
		}
		if argv, err := g.parseShebang(); err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(argv, tt.argv) {
			t.Errorf("expected %#v, got %#v", tt.argv, argv)
		}
	}
}

func TestReadShebang(t *testing.T) {
	dir, err := tempDir()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	g := &gola{}
	if _, err := g.readShebang(); err == nil {
		t.Error("expected error")
	}
	// empty file
	g.name = filepath.Join(dir, "a.py")
	if err := touch(g.name); err != nil {
		t.Fatal(err)
	}
	if _, err := g.readShebang(); err == nil {
		t.Error("expected error")
	}
	// shebang in file
	for _, tt := range []struct {
		data, shebang string
	}{
		{"#?/usr/bin/env python\n", ""},
		{"#!/usr/bin/env python", "#!/usr/bin/env python"},
		{"#!/usr/bin/env python\n", "#!/usr/bin/env python\n"},
	} {
		if err := file(g.name, tt.data); err != nil {
			t.Fatal(err)
		}
		if shebang, err := g.readShebang(); err != nil {
			t.Error(err)
		} else if shebang != tt.shebang {
			t.Errorf("expected %q, got %q", tt.shebang, shebang)
		}
	}
	// shebang in zipped file
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	if f, err := w.Create("__main__.py"); err != nil {
		t.Fatal(err)
	} else {
		f.Write([]byte("#!/usr/bin/env python\n"))
	}
	w.Close()
	g.name = filepath.Join(dir, "a.pyz")
	if err := file(g.name, buf.String()); err != nil {
		t.Fatal(err)
	}
	if shebang, err := g.readShebang(); err != nil {
		t.Error(err)
	} else if g, e := shebang, ""; g != e {
		t.Errorf("expected %q, got %q", e, g)
	}
	g.config.Dir = []string{"__main__.py"}
	if shebang, err := g.readShebang(); err != nil {
		t.Error(err)
	} else if g, e := shebang, "#!/usr/bin/env python\n"; g != e {
		t.Errorf("expected %q, got %q", e, g)
	}
}

func tempDir() (string, error) {
	return ioutil.TempDir("", "gola")
}

func file(name, data string) error {
	return ioutil.WriteFile(name, []byte(data), 0666)
}

func touch(name string) error {
	return ioutil.WriteFile(name, []byte{}, 0666)
}

func pushd(path string) (func() error, error) {
	wd, err := os.Getwd()
	popd := func() error {
		if err != nil {
			return err
		}
		return os.Chdir(wd)
	}
	return popd, os.Chdir(path)
}

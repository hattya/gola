# gola

A script launcher written in [Go](https://go.dev/).

[![GitHub Actions](https://github.com/hattya/gola/actions/workflows/ci.yml/badge.svg)](https://github.com/hattya/gola/actions/workflows/ci.yml)
[![Appveyor](https://ci.appveyor.com/api/projects/status/lchmkujc6phas1l5/branch/master?svg=true)](https://ci.appveyor.com/project/hattya/gola)
[![Codecov](https://codecov.io/gh/hattya/gola/branch/master/graph/badge.svg)](https://codecov.io/gh/hattya/gola)


## Installation

```console
$ go install github.com/hattya/gola@latest
```


## Usage

```console
$ gola [PATH] [OPTION]...
```


## Configuration

- `dir`  
  This is used for if the specified path is a directory or a zip file. It is
  redirected to a file that exists in the specified path.

  **Note**: The extension does not change.

- `map`  
  Described in the [Example configuration](#example-configuration).


### Search order for configuration files

1. `gola.json` - same location of the executable binary

2. `settings.json` - user's configuration directory

   - UNIX  
     `$XDG_CONFIG_HOME/gola/settings.json`

   - macOS  
     `~/Library/Application Support/gola/settings.json`

   - Windows  
     `%APPDATA%\gola\settings.json`


### Example configuration

```javascript
{
    "dir": [
        // Python can execute a directory or a zip file that
        // contains a __main__.py.
        "__main__.py"
    ],

    "map": {
        // This matches following cases:
        //   - #!/usr/bin/env python
        //   - #!/usr/bin/python
        //   - #!C:\Python27\python.exe
        //   - #!python.exe
        "python": {
            // This matches ".pyw" extension
            ".pyw": "C:\\Python27\\pythonw.exe",

            // This matches all extensions except mapped extensions
            "":     "C:\\Python27\\python.exe"
        },

        // This matches following cases:
        //   - #!/usr/bin/env python3.9
        //   - #!/usr/bin/env python3
        //   - #!/usr/bin/python3.9
        //   - #!/usr/bin/python3
        //   - #!python3.9.exe
        //   - #!python3.exe
        "python3": {
            ".pyw": "C:\\Python39\\pythonw.exe",
            "":     "C:\\Python39\\python.exe"
        }
    }
}
```

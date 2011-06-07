=====================================
gola: A script launcher written in Go
=====================================

Install
-------

* Go r57.1 -> http://golang.org/doc/install.html
* gola::

    $ git clone https://github.com/hattya/gola.git
    $ cd gola
    $ make

Usgae
-----

::

    $ gola [PATH] [OPTION]...

Configuration
-------------

dir
    The dir option is used for if the specified path is a directory or a zip
    file. It is redirected to a file that exists in the specified path.

    **Note**: The extension does not change.

map
    Described in the `Example configuration`_.

Search order for configuration files
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

1. gola.json - same location of the executable binary

on Windows

2. %APPDATA%\\gola\\settings.json

on Linux

2. $XDG_CONFIG_HOME/gola/settings.json
3. ~/.config/gola/settings.json

Example configuration
~~~~~~~~~~~~~~~~~~~~~

::

   {
       "dir": [
           // Python can execute a directory or a zip file that
           // contains a __main__.py.
           "__main__.py"
       ],

       "map": {
           // This matches follwing cases:
           //   - #!/usr/bin/env python
           //   - #!/usr/bin/python
           //   - #!C:\Python27\python.exe
           //   - #!python.exe
           "python": {
               // This matches ".pyw" extension
               ".pyw": "C:\\Python27\\pythonw.exe"

               // This matches all extensions except mapped extensions
               "":     "C:\\Python27\\python.exe"
           },

           // This matches follwing cases:
           //   - #!/usr/bin/env python3.2
           //   - #!/usr/bin/env python3
           //   - #!/usr/bin/python3.2
           //   - #!/usr/bin/python3
           //   - #!python3.2.exe
           //   - #!python3.exe
           "python3": {
               ".pyw": "C:\\Python32\\pythonw.exe"
               "":     "C:\\Python32\\python.exe"
           }
       }
   }

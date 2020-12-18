gola
====

A script launcher written in Go_.

.. image:: https://github.com/hattya/gola/workflows/CI/badge.svg
   :target: https://github.com/hattya/gola/actions?query=workflow:CI

.. image:: https://semaphoreci.com/api/v1/hattya/gola/branches/master/badge.svg
   :target: https://semaphoreci.com/hattya/gola

.. image:: https://ci.appveyor.com/api/projects/status/lchmkujc6phas1l5/branch/master?svg=true
   :target: https://ci.appveyor.com/project/hattya/gola

.. image:: https://codecov.io/gh/hattya/gola/branch/master/graph/badge.svg
   :target: https://codecov.io/gh/hattya/gola

.. _Go: https://golang.org/


Installation
------------

.. code:: console

   $ go get -u github.com/hattya/gola


Usage
-----

.. code:: console

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


Example configuration
~~~~~~~~~~~~~~~~~~~~~

.. code:: javascript

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

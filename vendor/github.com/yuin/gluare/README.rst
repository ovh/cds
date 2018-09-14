===============================================================================
gluare: Regular expressions for the GopherLua
===============================================================================

.. image:: https://travis-ci.org/yuin/gluare.svg
    :target: https://travis-ci.org/yuin/gluare

|

gluare is a regular expression library for the `GopherLua <https://github.com/yuin/gopher-lua>`_ .
gluare has almost the same API as Lua pattern match.

.. contents::
   :depth: 1

----------------------------------------------------------------
Installation
----------------------------------------------------------------

.. code-block:: bash
   
   go get github.com/yuin/gluare

----------------------------------------------------------------
Usage
----------------------------------------------------------------

.. code-block:: go

   import (
       "github.com/yuin/gopher-lua"
       "github.com/yuin/gluare"
   )
   
   L := lua.NewState()
   defer L.Close()
   L.PreloadModule("re", gluare.Loader)

----------------------------------------------------------------
Lua functions
----------------------------------------------------------------

`re.find` , `re.gsub`, `re.match`, `re.gmatch` are available. These functions have the same API as Lua pattern match. gluare uses the Go regexp package, so you can use regular expressions that are supported in the Go regexp package.

In addition, the following functions are defined:


**gluare.quote(s string) -> string**
  Arguments:
  
  =========================== ==============================================
  s string                    a string value to escape meta characters
  =========================== ==============================================
  
  
  Returns:
  
  =========================== ==============================================
  string                      escaped string
  =========================== ==============================================
  
  gluare.quote returns a string that quotes all regular expression metacharacters inside the given text.

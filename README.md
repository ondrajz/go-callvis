# go-callmap
go-callmap is a tool for visualizing the call graph of a Go program using the Graphviz dot language.

## Features
- focus specific package
- limit package path with prefix
- group functions by type/pkg

## Install
```
go get -u -v github.com/TrueFurby/go-callmap
```

## Legend

### nodes
Style | Description
--- | ---
*blue background* | focused package
*yellow background* | other packages
*dotted border* | anonymous func

### edges
Style | Description
--- | ---
*brown line* | outside focused package
*dashed line* | dynamic call
*empty arrow* | concurrent call
*empty circle* | deferred call

## Example
Here's example usage for [syncthing](https://github.com/syncthing/syncthing) program focusing *upgrade* package:
```
go-callmap -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```

![alt text](https://raw.githubusercontent.com/TrueFurby/go-callmap/master/images/syncthing.png)

and same using grouping by package:

```
go-callmap -sub pkg -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```

![alt text](https://raw.githubusercontent.com/TrueFurby/go-callmap/master/images/syncthing_pkg.png)

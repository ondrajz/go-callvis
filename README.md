# go-callvis
go-callvis is a tool to help visualize call graph of a Go program.

It runs [pointer analysis](https://godoc.org/golang.org/x/tools/go/pointer) to construct the call graph of the program and uses the data to generate output in [dot format](http://www.graphviz.org/content/dot-language), which can be rendered with graphviz tools.

## Features
- focus specific package
- limit package path with prefix
- group functions by type/pkg

## Install
```
go get -u -v github.com/TrueFurby/go-callvis
```

## Legend

### nodes
| Style | Description
| ----- | -----------
| *blue background* | focused package
| *yellow background* | other packages
| *dotted border* | anonymous func

### edges
| Style | Description
| ----- | -----------
| *brown line* | outside focused package
| *dashed line* | dynamic call
| *empty arrow* | concurrent call
| *empty circle* | deferred call

## Example
Here's example usage for [syncthing](https://github.com/syncthing/syncthing) program focusing *upgrade* package:

```
go-callvis -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```

![syncthing example output](images/syncthing.png)

and same using grouping by package:

```
go-callvis -sub pkg -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```

![syncthing example output](images/syncthing_pkg.png)

# go-callvis

![example](images/main.png)

go-callvis is a development tool to help visualize call graph of your Go program.

Intended purpose of this tool is to show overview of your code's structure by visually representing call graph and type relations. This is especially useful in larger projects where the complexity of the code rises.

### Features

- **focus** specific **package** in program
- **limit** to include only packages containing **prefix**
- **ignore** multiple packages containing **prefix**
- **group** functions by **types/packages**

### Requirements

* [Go](https://golang.org/dl/)
* [GraphViz](http://www.graphviz.org/Download..php)

### Installation

`go get -u -v github.com/TrueFurby/go-callvis`

## Legend

Element             | Style    | Represents
------------------: | :------: | -----------
__node background__ |  _blue_  | focused package
                    | _yellow_ | non-focused packages
    __node border__ |  _bold_  | exported func
                    | _dotted_ | anonymous func
      __edge line__ | _brown_  | outside focused package
                    | _dashed_ | dynamic call
     __edge arrow__ | _empty_  | concurrent call
    __edge circle__ | _empty_  | deferred call

## Examples

Here are usage examples for [syncthing](https://github.com/syncthing/syncthing) program.

+ **focusing package** _upgrade_

  ![syncthing example output](images/syncthing.png)
```
go-callvis -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```

+ with **grouping by packages**

  ![syncthing example output pkg](images/syncthing_pkg.png)
```
go-callvis -sub pkg -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```

+ and **ignoring package** _logger_

  ![syncthing example output ignore](images/syncthing_ignore.png)
```
go-callvis -ignore github.com/syncthing/syncthing/lib/logger -sub pkg -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```

## Flags

```
Usage of go-callvis:
  -focus string
    	focus package name
  -ignore string
    	ignore package path
  -limit string
    	limit package path
  -sub string
    	subgraph by [type, pkg]
```

## How it works

It runs [pointer analysis](https://godoc.org/golang.org/x/tools/go/pointer) to construct the call graph of the program and uses the data to generate output in [dot format](http://www.graphviz.org/content/dot-language), which can be rendered with graphviz tools.

## Roadmap

Ideal goal of this project is to make web app that would locally store the call graph data and then provide quick access of the call graphs for any package of your dependency tree. At first it would show an interactive map of overall dependencies between packages and then by selecting particular package it would show the call graph and provide various options to alter the output dynamically.

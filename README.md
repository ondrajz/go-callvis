<p align="center"><img src="images/gopher.png" alt="gopher"></p>

<p align="center">
  <a href="https://github.com/TrueFurby/go-callvis/releases"><img src="https://img.shields.io/github/release/truefurby/go-callvis.svg" alt="Github release"></a> <a href="https://travis-ci.org/TrueFurby/go-callvis"><img src="https://travis-ci.org/TrueFurby/go-callvis.svg?branch=master" alt="Build status"></a> <a href="https://gophers.slack.com/archives/go-callvis"><img src="https://img.shields.io/badge/gophers%20slack-%23go--callvis-ff69b4.svg" alt="Slack channel"></a>
</p>

# <div align="center">go-callvis</div>

<p align="center"><b>go-callvis</b> is a development tool to help visualize call graph of your Go program using Graphviz's dot format.</p>

[![main](images/main.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/main.png)

## Introduction

Intended purpose of this tool is to provide a visual overview of function calls of your program by using call graph, package and type relations. This is especially useful in larger projects where the complexity of the code rises or when you are trying to understand someone else's code.

### Features

- focus specific package in a program
- group funcs by packages and/or types
- limit packages to custom prefix path
- ignore packages containing custom prefix

#### How it works

It runs [pointer analysis](https://godoc.org/golang.org/x/tools/go/pointer) to construct the call graph of the program and uses the data to generate output in [dot format](http://www.graphviz.org/content/dot-language), which can be rendered with Graphviz tools.

## Get started

### Requirements

- [Go](https://golang.org/dl/)
- [Graphviz](http://www.graphviz.org/Download..php)

### Install

Use the following commands to install:

```
go get -u github.com/TrueFurby/go-callvis
cd $GOPATH/src/github.com/TrueFurby/go-callvis
make
```

### Usage

```
go-callvis [OPTIONS] <main pkg>

Options:
  -focus string
        Focus package with import path or name (default: main).
  -limit string
        Limit package path to prefix.
  -group string
        Grouping functions by [pkg, type] (separate multiple by comma).
  -ignore string
        Ignore package paths with prefix (separate multiple by comma).
  -tests
        Include test code.
  -debug
        Enable verbose log.
  -version
        Show version and exit.
```

## Legend

### Packages & Types

###### Presented as subgraphs (clusters).

##### Packages
- *normal corners*
- *label on the top*

##### Types
- *rounded corners*
- *label on the bottom*

Represents  | Style
----------: | :-------------
  `focused` | _blue color_
   `stdlib` | _green color_
    `other` | _yellow color_

### Functions

###### Presented as nodes.

Represents   | Style
-----------: | :--------------
  `exported` | _bold border_
`unexported` | _normal border_
 `anonymous` | _dotted border_

### Calls

###### Presented as edges.

Represents   | Style
-----------: | :-------------
  `internal` | _black color_
  `external` | _brown color_
    `static` | _solid line_
   `dynamic` | _dashed line_
   `regular` | _simple arrow_
`concurrent` | _arrow with empty circle_
  `deferred` | _arrow with empty diamond_

## Examples

Here is an example for the project [syncthing](https://github.com/syncthing/syncthing).

[![syncthing example](images/syncthing.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/syncthing.png)

```
go-callvis -focus upgrade -group pkg,type -limit github.com/syncthing/syncthing -ignore github.com/syncthing/syncthing/lib/logger github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```

###### You can find more examples in the folder [examples](examples).

### Community

Join the channel [#go-callvis](https://gophers.slack.com/archives/go-callvis) at [gophers.slack.com](http://gophers.slack.com).

> *Are you not a member yet?* [Get invitation](https://gophersinvite.herokuapp.com).

#### How to contribute

###### *Did you find any bugs or have some suggestions?*

Feel free to open [new issue](https://github.com/TrueFurby/go-callvis/issues/new) or start discussion in the slack channel.

#### Known Issues

+ **each execution takes a lot of time, because currently:**
  - the call graph is always generated for the entire program
  - there is yet no caching of call graph data

---

#### Roadmap

###### :boom: The *interactive tool* described below has been published as a *separate project* called [goexplorer](https://github.com/TrueFurby/goexplorer).

> Ideal goal of this project is to make web app that would locally store the call graph data and then provide quick access of the call graphs for any package of your dependency tree. At first it would show an interactive map of overall dependencies between packages and then by selecting particular package it would show the call graph and provide various options to alter the output dynamically.

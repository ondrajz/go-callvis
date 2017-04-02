<p align="center"><img src="images/gopher.png" alt="gopher"></p>
  
<h1 align="center">go-callvis</h1>

<p align="center">
  <a href="https://github.com/TrueFurby/go-callvis/releases"><img src="https://img.shields.io/github/release/truefurby/go-callvis.svg" alt="Github release"></a>
  <a href="https://travis-ci.org/TrueFurby/go-callvis"><img src="https://travis-ci.org/TrueFurby/go-callvis.svg?branch=master" alt="Build status"></a>
  <a href="https://gophers.slack.com/archives/go-callvis"><img src="https://img.shields.io/badge/gophers%20slack-%23go--callvis-ff69b4.svg" alt="Slack channel"></a>
</p>

<p align="center"><b>go-callvis</b> is a development tool to help visualize call graph of your Go program using Graphviz's dot format.</p>

---

## Introduction

Purpose of this tool is to provide a visual overview of your program by using the data from call graph and its relations with packages and types. This is especially useful in larger projects where the complexity of the code rises or when you are just simply trying to understand code structure of somebody else.

### Features

- focus specific package in a program
- group functions by package and methods by type
- limit packages to custom path prefixes
- ignore packages containing path prefixes
- omit calls from/to std packages

### Output preview

[![main](images/main.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/main.png)

> Check out the [source code](examples/main) for the above image.

### How it works

It runs [pointer analysis](https://godoc.org/golang.org/x/tools/go/pointer) to construct the call graph of the program and uses the data to generate output in [dot format](http://www.graphviz.org/content/dot-language), which can be rendered with Graphviz tools.

## Reference guide

Here you can find descriptions for all possible kinds of calls and groups.

### Packages / Types

###### Represented as subgraphs (clusters) in output.

**Packages**
- _**normal** corners_
- _label on the **top**_

**Types**
- _**rounded** corners_
- _label on the **bottom**_

Represents  | Style
----------: | :-------------
`focused`   | _**blue** color_
`stdlib`    | _**green** color_
`other`     | _**yellow** color_

### Functions / Methods

###### Represented as nodes in output.

Represents   | Style
-----------: | :--------------
`exported`   | _**bold** border_
`unexported` | _**normal** border_
`anonymous`  | _**dotted** border_

### Calls

###### Represented as edges in output.

Represents   | Style
-----------: | :-------------
`internal`   | _**black** color_
`external`   | _**brown** color_
`static`     | _**solid** line_
`dynamic`    | _**dashed** line_
`regular`    | _**simple** arrow_
`concurrent` | _arrow with **circle**_
`deferred`   | _arrow with **diamond**_

## Quick start

#### Requirements

- [Go](https://golang.org/dl/)
- [Graphviz](http://www.graphviz.org/Download..php)

### Installation

```sh
go get -u github.com/TrueFurby/go-callvis
cd $GOPATH/src/github.com/TrueFurby/go-callvis && make
```

### Usage

`go-callvis [OPTIONS] <main pkg> | dot -Tpng -o output.png`

### Options

```
-focus string
      Focus package with import path or name. (default: main)
-limit string
      Limit package paths to prefix. (separate multiple by comma)
-group string
      Grouping functions by [pkg, type]. (separate multiple by comma)
-ignore string
      Ignore package paths with prefix. (separate multiple by comma)
-nostd
      Omit calls from/to std packages.
-minlen uint
      Minimum edge length (for wider output). (default: 2)
-nodesep float
      Minimum space between two adjacent nodes in the same rank (for taller output). (default: 0.35)
```

## Examples

Here is an example for the project [syncthing](https://github.com/syncthing/syncthing).

[![syncthing example](images/syncthing.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/syncthing.png)

> Check out [more examples](examples) and used command options.

## Community

Join [#go-callvis](https://gophers.slack.com/archives/go-callvis) channel at [gophers.slack.com](http://gophers.slack.com).

> *Not a member yet?* [Get invitation](https://gophersinvite.herokuapp.com).

### How to help

###### Did you find any bugs or have some suggestions?
Feel free to open [new issue](https://github.com/TrueFurby/go-callvis/issues/new) or start discussion in the slack channel.

###### Do you want to contribute to the development?
Fork the project and do a pull request. [Here](https://github.com/TrueFurby/go-callvis/projects/1) you can find the state of features.

### Known Issues

###### Each execution takes a lot of time, because currently:
- the call graph is always generated for the entire program
- there is yet no caching of call graph data

---

### Roadmap

#### The *interactive tool* described below has been published as a *separate project* called [goexplorer](https://github.com/TrueFurby/goexplorer)! :boom:

> Ideal goal of this project is to make web app that would locally store the call graph data and then provide quick access of the call graphs for any package of your dependency tree. At first it would show an interactive map of overall dependencies between packages and then by selecting particular package it would show the call graph and provide various options to alter the output dynamically.

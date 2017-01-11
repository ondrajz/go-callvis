go-callvis [![Go Report Card](https://goreportcard.com/badge/github.com/TrueFurby/go-callvis)](https://goreportcard.com/report/github.com/TrueFurby/go-callvis) [![Build Status](https://travis-ci.org/TrueFurby/go-callvis.svg?branch=master)](https://travis-ci.org/TrueFurby/go-callvis)
==========

**go-callvis** is a development tool to help visualize call graph of your Go program using Graphviz's dot format.

![example](images/example.png)
> check [source code](example) for this example

Intended purpose of this tool is to provide a visual overview of your program's source code structure by using call graph and type relations. This is especially useful in larger projects where the complexity of the code rises.

## Features

- focus specific package in a program
- group functions by types or packages
- limit packages to custom prefix path
- ignore packages containing custom prefix

### How it works

It runs [pointer analysis](https://godoc.org/golang.org/x/tools/go/pointer) to construct the call graph of the program and uses the data to generate output in [dot format](http://www.graphviz.org/content/dot-language), which can be rendered with Graphviz tools.

## Installation

### Requirements

- [Go](https://golang.org/dl/)
- [Graphviz](http://www.graphviz.org/Download..php)

### Install

Use the following command to install:

```
go get -u -v github.com/TrueFurby/go-callvis
```

### Usage

```
go-callvis [OPTIONS] <main pkg>

Options:
  -focus string
        Focus package with name (default: main)
  -limit string
        Limit package path to prefix
  -group string
        Grouping by [type, pkg]
  -ignore string
        Ignore package paths with prefix (separated by comma)
  -test bool
        Include test code
```

## Legend

#### Packages

Type        | Style          |                   Example
----------: | :------------- | :-----------------------------------------:
**focused** | _blue color_   |    ![focused](images/legend_focused.png)
  **other** | _yellow color_ | ![nonfocused](images/legend_nonfocused.png)

#### Functions (_nodes_)

Type           | Style           |                  Example
-------------: | :-------------- | :----------------------------------------:
  **exported** | _bold border_   |  ![exported](images/legend_exported.png)
**unexported** | _normal border_ | ![anonymous](images/legend_unexported.png)
 **anonymous** | _dotted border_ | ![anonymous](images/legend_anonymous.png)

#### Calls (_edges_)

Type           | Style          |                   Example
-------------: | :------------- | :-----------------------------------------:
  **internal** | _black color_  |   ![outside](images/legend_internal.png)
  **external** | _brown color_  |   ![outside](images/legend_external.png)
   **dynamic** | _dashed line_  |    ![dynamic](images/legend_dynamic.png)
**concurrent** | _empty arrow_  | ![concurrent](images/legend_concurrent.png)
  **deferred** | _empty circle_ |   ![deferred](images/legend_deferred.png)

## Examples

Here are usage examples for [syncthing](https://github.com/syncthing/syncthing) program.

### Focusing package _upgrade_

![syncthing example output](images/syncthing_focus.png)

```
go-callvis -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```

--------------------------------------------------------------------------------

### Grouping by _packages_

![syncthing example output pkg](images/syncthing_group.png)

```
go-callvis -group pkg -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```

--------------------------------------------------------------------------------

### Ignoring package _logger_

![syncthing example output ignore](images/syncthing_ignore.png)

```
go-callvis -ignore github.com/syncthing/syncthing/lib/logger -group pkg -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```

## Roadmap

Ideal goal of this project is to make web app that would locally store the call graph data and then provide quick access of the call graphs for any package of your dependency tree. At first it would show an interactive map of overall dependencies between packages and then by selecting particular package it would show the call graph and provide various options to alter the output dynamically.

## Known Issues

**execution takes a lot of time (~5s), because currently:**

- the call graph is always generated for the entire program
- there is yet no caching of call graph data

## Community

Join the [#go-callvis](https://gophers.slack.com/archives/go-callvis) channel at [gophers.slack.com](http://gophers.slack.com)

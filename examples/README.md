# Examples

This document contains examples of various projects from Github.

#### Projects
  - [Syncthing](https://github.com/syncthing/syncthing)
  - [Docker](https://github.com/docker/docker)
  - [Travis CI Worker](https://github.com/travis-ci/worker)


## Syncthing

[![syncthing example](../images/syncthing.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/syncthing.png)

```sh
# Setup
go get -u github.com/syncthing/syncthing/
cd $GOPATH/src/github.com/syncthing/syncthing
./build.sh
```

```sh
go-callvis -focus upgrade -group pkg,type -limit github.com/syncthing/syncthing -ignore github.com/syncthing/syncthing/lib/logger github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```
---

### Focusing package _upgrade_

[![syncthing example output](../images/syncthing_focus.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/syncthing_focus.png)

```sh
go-callvis -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing_focus.png
```
---

### Grouping by _packages_

[![syncthing example output pkg](../images/syncthing_group.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/syncthing_group.png)

```sh
go-callvis -focus upgrade -group pkg -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing_group.png
```
---

### Ignoring package _logger_

[![syncthing example output ignore](../images/syncthing_ignore.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/syncthing_ignore.png)

```sh
go-callvis -focus upgrade -group pkg -ignore github.com/syncthing/syncthing/lib/logger -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing_ignore.png
```
---

## Docker

[![docker example](../images/docker.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/docker.png)

```sh
go-callvis -limit github.com/docker/docker -ignore github.com/docker/docker/vendor github.com/docker/docker/cmd/docker | dot -Tpng -o docker.png
```
---

## Travis CI Worker

[![travis-example](../images/travis_thumb.jpg)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/travis.jpg)

```sh
go-callvis -minlen 3 -nostd -group type,pkg -focus worker -limit github.com/travis-ci/worker -ignore github.com/travis-ci/worker/vendor github.com/travis-ci/worker/cmd/travis-worker > travis.dot && dot -Tsvg -o travis.svg travis.dot && exo-open travis.svg
```

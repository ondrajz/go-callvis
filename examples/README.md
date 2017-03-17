# Examples
## [Syncthing](https://github.com/syncthing/syncthing)

[![syncthing example](../images/syncthing.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/syncthing.png)

```
go-callvis -focus upgrade -group pkg,type -limit github.com/syncthing/syncthing -ignore github.com/syncthing/syncthing/lib/logger github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing.png
```
---

### Focusing package _upgrade_

[![syncthing example output](../images/syncthing_focus.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/syncthing_focus.png)

```
go-callvis -focus upgrade -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing_focus.png
```
---

### Grouping by _packages_

[![syncthing example output pkg](../images/syncthing_group.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/syncthing_group.png)

```
go-callvis -focus upgrade -group pkg -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing_group.png
```
---

### Ignoring package _logger_

[![syncthing example output ignore](../images/syncthing_ignore.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/syncthing_ignore.png)

```
go-callvis -focus upgrade -group pkg -ignore github.com/syncthing/syncthing/lib/logger -limit github.com/syncthing/syncthing github.com/syncthing/syncthing/cmd/syncthing | dot -Tpng -o syncthing_ignore.png
```
---

## [Docker](https://github.com/docker/docker)

[![docker example](../images/docker.png)](https://raw.githubusercontent.com/TrueFurby/go-callvis/master/images/docker.png)

```
go-callvis -limit github.com/docker/docker -ignore github.com/docker/docker/vendor github.com/docker/docker/cmd/docker | dot -Tpng -o docker.png
```
---

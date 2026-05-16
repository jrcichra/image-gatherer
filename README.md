# image-gatherer

Watches a list of container image repositories and persists their latest tags to a file or Git repo. Useful for GitOps workflows where you want to automatically track upstream image versions without manual updates.

## How it works

On each run, image-gatherer resolves the latest tag for every configured container using a pluggable strategy, then writes the results via a configurable output plugin. Runs on an interval until stopped.

## Plugins

### Input plugins

**`semver`** — queries the container registry for all tags, parses them as semantic versions, and returns the highest one. Supports filtering out unwanted tags (e.g. release candidates) via regex.

**`git`** — clones a Git repository, reads the HEAD commit hash of a branch, then finds a matching container image tag (e.g. `sha-abc1234`). Walks parent commits if the HEAD has no matching image yet.

### Output plugins

**`file`** — writes results to a local YAML file.

**`git`** — clones a Git repo, writes the YAML file, commits, and pushes. No-ops if nothing changed. Useful for storing results in a GitOps repo.

## Configuration

```yaml
containers:
  <friendly-name>:
    container: <registry>/<repo>   # full image reference
    plugin: semver | git           # which input plugin to use
    pin: <tag>                     # optional: skip lookup and use this value
    options:
      <key>: <value>               # plugin-specific options (see below)

output:
  plugin: file | git
  options:
    <key>: <value>
```

### Semver input options

| Option | Description |
|---|---|
| `ignore_regexes` | Comma-separated list of regexes. Tags whose semver string matches any regex are excluded. Example: `-rc\d+,-alpha` |

### Git input options

| Option | Required | Description |
|---|---|---|
| `url` | yes | Git repository URL to clone |
| `branch` | yes | Branch name to read HEAD from |
| `username_env` | no | Env var name holding the HTTP username |
| `password_env` | no | Env var name holding the HTTP password or token |
| `ssh` | no | Set to `true` to use SSH key auth |
| `ssh_key_path` | no | Path to SSH private key (default: `~/.ssh/id_rsa`) |

### Git output options

| Option | Required | Description |
|---|---|---|
| `url` | yes | Git repository URL to clone and push to |
| `branch` | yes | Branch to commit to |
| `filename` | yes | Filename to write inside the repo |
| `username_env` | no | Env var name holding the HTTP username |
| `password_env` | no | Env var name holding the HTTP password or token |
| `ssh` | no | Set to `true` to use SSH key auth |
| `ssh_key_path` | no | Path to SSH private key (default: `~/.ssh/id_rsa`) |
| `commit_author_name` | no | Git commit author name (default: `Image Gatherer`) |
| `commit_author_email` | no | Git commit author email (default: `imagegatherer@jrcichra.dev`) |

### File output options

| Option | Required | Description |
|---|---|---|
| `name` | yes | Path to write the output YAML file |

## Example config

```yaml
containers:
  # Track latest stable busybox via semver
  busybox:
    container: docker.io/library/busybox
    plugin: semver

  # Skip release candidates
  gotosocial:
    container: docker.io/superseriousbusiness/gotosocial
    plugin: semver
    options:
      ignore_regexes: "-rc\\d+"

  # Track a container image by matching the Git repo's HEAD commit hash
  my-app:
    container: ghcr.io/myorg/my-app
    plugin: git
    options:
      url: https://github.com/myorg/my-app.git
      branch: main
      username_env: GIT_USER
      password_env: GIT_TOKEN

  # Pin a specific version and skip the lookup
  stable-dep:
    container: docker.io/library/nginx
    plugin: semver
    pin: "1.27.0"

output:
  plugin: git
  options:
    url: https://github.com/myorg/gitops-repo.git
    branch: main
    filename: versions.yaml
    username_env: GIT_USER
    password_env: GIT_TOKEN
    commit_author_name: "image-gatherer bot"
    commit_author_email: "bot@myorg.com"
```

## Running

```sh
# With defaults (config.yaml, 5 minute interval)
./image-gatherer

# Custom config and interval
./image-gatherer -config /etc/image-gatherer/config.yaml -interval 10m
```

### Docker

```sh
docker run -v $(pwd)/config.yaml:/config.yaml ghcr.io/jrcichra/image-gatherer
```

### Docker Compose

```yaml
services:
  image-gatherer:
    image: ghcr.io/jrcichra/image-gatherer:latest
    volumes:
      - ./config.yaml:/config.yaml
    environment:
      GIT_USER: myuser
      GIT_TOKEN: ghp_...
    restart: unless-stopped
```

## Output format

Results are written as a YAML map of friendly name to fully-qualified image reference:

```yaml
containers:
  busybox: docker.io/library/busybox:1.37.0
  gotosocial: docker.io/superseriousbusiness/gotosocial:0.17.3
  my-app: ghcr.io/myorg/my-app:sha-abc1234
  stable-dep: docker.io/library/nginx:1.27.0
```

## Container registry authentication

Image-gatherer uses Docker's credential store automatically. Run `docker login <registry>` before starting, or mount `~/.docker/config.json` into the container.

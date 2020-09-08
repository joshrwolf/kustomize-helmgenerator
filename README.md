# kustomize-helmgenerator

A simple _exec_ plugin for Kustomize written in `go` to declaratively define `helm template` outputs as a Kustomize generator.

`HelmChart` is purposely not written as a Kustomize `go` plugin to prevent integration difficulties with various versions of Kustomize.

## Examples

The `HelmChart` API is heavily inspired by the Helm portion of the `argocd-cm` api [here](https://argoproj.github.io/argo-cd/operator-manual/argocd-cm.yaml).  The full spec is located below:

```yaml
apiVersion: wolfs.io/v1beta1
kind: HelmChart
metadata:
  # Chart's release name ({{ .Release.Name }}) mathces HelmChart name
  name: mocha
  # Chart's release namespace ({{ .Release.Namespace }}) matches HelmChart namespace
  namespace: dog

# Chart sourced from external git repository
chart:
  git: https://<path/to/git/repo>.git
  ref: <branch>
  path: relative/path

# Chart sourced from external helm repository
chart:
  repository: https://<path/to/helm/repo>
  name: <chart name>
  version: <chart version>

# Chart sourced locally
chart:
  path: relative/path

# Ordered helm value imports (values will be overrided in order)
valueFiles:
  - testdata/values-base.yaml
  - testdata/values-prod.yaml

# Generic map of values (will take precedence over value files above)
values:
  image:
    repository: donkers

sopsValueFiles:
  - path/to/encrypted.enc.yaml
```

## Installation

Download `HelmChart` either from source or from the [Github release page](https://github.com/joshrwolf/kustomize-helmgenerator/releases).

Like any Kustomize plugin, an `XDG_CONFIG_HOME` environment variable must be set before installing.  See [here](https://github.com/kubernetes-sigs/kustomize/tree/master/docs/plugins) for more info.

### From Source

```bash
# Clone the repo
git clone https://github.com/joshrwolf/kustomize-helmgenerator.git
cd kustomize-helmgenerator

# Build the plugin (tested with go 1.14) and move it to the appropriate $XDG_CONFIG_HOME
make install
```

### From Release

Executables are provided for `amd64` on `darwin`, `linux`, and `windows`.  Download it by specifying your appropriate arch, platform, and version:

```bash
VERSION=0.1.0 PLATFORM=linux ARCH=amd64 curl -Lo HelmChart https://github.com/joshrwolf/kustomize-helmgenerator/releases/download/v${VERSION}/HelmChart_${VERSION}_${PLATFORM}_${ARCH}
mkdir -p ${XDG_CONFIG_HOME}/kustomize/plugin/wolfs.io/v1beta1/helmchart
chmod +x HelmChart
mv HelmChart ${XDG_CONFIG_HOME}/kustomize/plugin/wolfs.io/v1beta1/helmchart/
```

# kustomize-helmgenerator

A simple _exec_ plugin for Kustomize written in `go` to declaratively define `helm template` outputs as a Kustomize generator.

`HelmGenerator` is purposely not written as a Kustomize `go` plugin to prevent integration difficulties with various versions of Kustomize.

## Examples

The `HelmGenerator` API is heavily inspired by the Helm portion of the `argocd-cm` api [here](https://argoproj.github.io/argo-cd/operator-manual/argocd-cm.yaml).  An example of a feature rich generator is below:

```yaml
apiVersion: wolfs.io/v1beta1
kind: HelmGenerator
metadata:
  name: helmGenerator

# Charge release name
releaseName: dog

# Path to a chart directory or .tgz of a chart (only local chart paths supported right now)
chartPath: testdata/mocha/

# Namespace to deploy chart to
namespace: coco

# Ordered helm value imports (values will be overrided in order)
valueFiles:
  - testdata/values-base.yaml
  - testdata/values-prod.yaml

# Generic map of values (will overwrite valuesFiles above)
values: |
  image:
    repository: donkers
```

## Installation

Download `HelmGenerator` either from source or from the [Github release page](https://github.com/joshrwolf/kustomize-helmgenerator/releases).

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
VERSION=0.1.0 PLATFORM=linux ARCH=amd64 curl -Lo HelmGenerator https://github.com/joshrwolf/kustomize-helmgenerator/releases/download/v${VERSION}/HelmGenerator_${VERSION}_${PLATFORM}_${ARCH}
mkdir -p ${XDG_CONFIG_HOME}/kustomize/plugin/wolfs.io/v1beta1/helmgenerator
chmod +x HelmGenerator
mv HelmGenerator ${XDG_CONFIG_HOME}/kustomize/plugin/wolfs.io/v1beta1/helmgenerator/
```

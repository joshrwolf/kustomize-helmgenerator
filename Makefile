export GO111MODULE=on

PLUGIN_DIR := ${XDG_CONFIG_HOME}/kustomize/plugin/p1.dsop.io/v1beta1/helmgenerator

HelmGenerator: HelmGenerator.go
	go build -o $@ $<

.PHONY: install
install: HelmGenerator
	@mkdir -p ${PLUGIN_DIR}
	mv HelmGenerator ${PLUGIN_DIR}

.PHONY: release
release:
	goreleaser --rm-dist --skip-publish

.PHONY: publish-release
publish-release:
	goreleaser --rm-dist

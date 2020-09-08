export GO111MODULE=on

PLUGIN_DIR := ${XDG_CONFIG_HOME}/kustomize/plugin/wolfs.io/v1beta1/helmchart

HelmChart: HelmChart.go
	go build -o $@ $<

.PHONY: install
install: HelmChart
	@mkdir -p ${PLUGIN_DIR}
	mv HelmChart ${PLUGIN_DIR}

.PHONY: release
release:
	goreleaser --rm-dist --skip-publish

.PHONY: publish-release
publish-release:
	goreleaser --rm-dist

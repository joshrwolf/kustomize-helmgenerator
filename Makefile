export GO111MODULE=on

HelmGenerator: HelmGenerator.go
	go build -o $@ $<

.PHONY: release
release:
	goreleaser --rm-dist --skip-publish

.PHONY: publish-release
publish-release:
	goreleaser --rm-dist

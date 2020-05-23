export GO111MODULE=on

HelmGenerator: HelmGenerator.go
	go build -o $@ $<

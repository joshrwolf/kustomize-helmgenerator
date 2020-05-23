package main

import (
	"testing"
)

func Test_processHelmGenerator(t *testing.T) {
	type args struct {
		fn string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"HelmGenerator",
			args{"testdata/generator.yaml"},
			`---
# Source: mocha/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: mocha
spec:
  type: ClusterIP
  ports:
  - port: 80
	targetPort: 80
	protocol: TCP
	name: http
---
# Source: mocha/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mocha
spec:
  replicas: 99
  template:
	spec:
	  containers:
		- name: mocha
		  image: "donkers:1.16.0"
		  imagePullPolicy: Always
		  ports:
			- name: http
			  containerPort: 80
			  protocol: TCP
`,
		false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processHelmGenerator(tt.args.fn)
			if (err != nil) != tt.wantErr {
				t.Errorf("processHelmGenerator() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("processHelmGenerator() got = %v, want %v", got, tt.want)
			}
		})
	}
}
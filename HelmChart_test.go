package main

import (
	"context"
	"encoding/json"
	"fmt"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"testing"
)

const (
	simpleChartPath = "testdata/charts/chart"
	releaseName = "mocha"
	releaseNamespace = "dog"
)

func TestHelmChart_simpleTemplate(t *testing.T) {
	ctx := context.Background()

	chrt, err := loader.Load(simpleChartPath)
	if err != nil {
		t.Errorf("Failed to load the sample chart: %v", err)
	}

	type fields struct {
		Chart          Chart
		ValueFiles     []string
		Values         map[string]interface{}
		SopsValueFiles []string
	}

	var tv map[string]interface{}
	err = json.Unmarshal([]byte(`{"image": {"tag": "latest"}}`), &tv)

	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "Test default chart values",
			fields: fields{
				Chart: Chart{
					Path: simpleChartPath,
				},
			},
			want: fmt.Sprintf(`---
# Source: mocha/templates/pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: %v
  namespace: %v
spec:
  containers:
    - name: container
      image: rancher/rancher:stable

---
apiVersion: batch/v1
kind: Job
metadata:
  name: %v
  namespace: %v
  annotations:
    "helm.sh/hook": post-install
    "helm.sh/hook-weight": "-5"
    "helm.sh/hook-delete-policy": hook-succeeded
spec:
  template:
    spec:
      containers:
        - name: post-install-job
          image: rancher/pause:stable`, releaseName, releaseNamespace, releaseName, releaseNamespace),
		},
		{
			name: "Test values as map",
			fields: fields{
				Chart: Chart{
					Path: simpleChartPath,
				},
				Values: tv,
			},
			want: fmt.Sprintf(`---
# Source: mocha/templates/pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: %v
  namespace: %v
spec:
  containers:
    - name: container
      image: rancher/rancher:latest

---
apiVersion: batch/v1
kind: Job
metadata:
  name: %v
  namespace: %v
  annotations:
    "helm.sh/hook": post-install
    "helm.sh/hook-weight": "-5"
    "helm.sh/hook-delete-policy": hook-succeeded
spec:
  template:
    spec:
      containers:
        - name: post-install-job
          image: rancher/pause:stable`, releaseName, releaseNamespace, releaseName, releaseNamespace),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &HelmChart{
				TypeMeta:       TypeMeta{APIVersion: apiVersion, Kind: kind},
				ObjectMeta:     ObjectMeta{Name: releaseName, Namespace: releaseNamespace},
				Chart:          tt.fields.Chart,
				ValueFiles:     tt.fields.ValueFiles,
				Values:         tt.fields.Values,
				SopsValueFiles: tt.fields.SopsValueFiles,
			}

			got, err := c.template(ctx, chrt)
			if (err != nil) != tt.wantErr {
				t.Errorf("template() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("template() got = %v, want %v", got, tt.want)
			}
		})
	}
}


func TestHelmChart_template(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		Chart          Chart
		ValueFiles     []string
		Values         map[string]interface{}
		SopsValueFiles []string
	}
	type args struct {
		ctx context.Context
		ch  *chart.Chart
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Test from local full tarball",
			fields: fields{
				Chart: Chart{
					Path: "testdata/charts/chart-with-compressed-dependencies-2.1.8.tgz",
				},
			},
			args: args{},
			want: "",
		},
		{
			name: "Test from local chart with missing dependencies",
			fields: fields{
				Chart: Chart{
					Path: "testdata/charts/chart-missing-deps",
				},
			},
			args: args{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &HelmChart{
				TypeMeta:       TypeMeta{APIVersion: apiVersion, Kind: kind},
				ObjectMeta:     ObjectMeta{Name: "mocha", Namespace: "dog"},
				Chart:          tt.fields.Chart,
				ValueFiles:     tt.fields.ValueFiles,
				Values:         tt.fields.Values,
				SopsValueFiles: tt.fields.SopsValueFiles,
			}

			chrt, err := loadAndUpdate(ctx, tt.fields.Chart.Path)
			if err != nil {
				t.Errorf("failed to load chart: %v", err)
			}
			got, err := c.template(ctx, chrt)
			if (err != nil) != tt.wantErr {
				t.Errorf("template() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("template() got = %v, want %v", got, tt.want)
			}
		})
	}
}
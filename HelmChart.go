package main

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"go.mozilla.org/sops/v3/decrypt"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const apiVersion = "p1.dsop.io/v1beta1"
const kind = "HelmChart"

type kvMap map[string]string

type TypeMeta struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
}

type ObjectMeta struct {
	Name        string `json:"name" yaml:"name"`
	Namespace   string `json:"namespace" yaml:"namespace"`
	Labels      kvMap  `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations kvMap  `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

type Chart struct {
	Git string `json:"git,omitempty" yaml:"git,omitempty"`
	Ref string `json:"ref,omitempty" yaml:"ref,omitempty"`
	Path string `json:"path,omitempty" yaml:"path,omitempty"`
	Repository string `json:"repository,omitempty" yaml:"repository,omitempty"`
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
}

type HelmChart struct {
	TypeMeta   `json:",inline" yaml:",inline"`
	ObjectMeta `json:"metadata" yaml:"metadata"`

	Chart Chart `json:"chart" yaml:"chart"`

	// List of value files to use, will be used in order
	ValueFiles []string `json:"valueFiles,omitempty" yaml:"valueFiles,omitempty"`

	// Values is a generic map of values, takes precedence over ValueFiles and SopsValueFiles
	Values map[string]interface{} `json:"values,omitempty" yaml:"values,omitempty"`

	// SopsValueFiles is a generic list of values files encrypted via sops
	SopsValueFiles []string `json:"sopsValueFiles,omitempty" yaml:"sopsValueFiles,omitempty"`
}

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: HelmChart FILE")
		os.Exit(1)
	}

	ctx := context.Background()

	hc, err := unmarshalHelmChart(os.Args[1])

	var chrt *chart.Chart
	// Figure out what we're dealing with
	if hc.Chart.Repository != "" {
		// Load from chart repository
		chrt, err = pull(ctx, hc.Chart.Repository, hc.Chart.Name, hc.Chart.Version)
	} else if hc.Chart.Git != "" {
		// Load from git repo
		chrt, err = clone(ctx, hc.Chart.Git, hc.Chart.Ref, hc.Chart.Path)

	} else {
		// Load from a local path
		chrt, err = loadAndUpdate(ctx, hc.Chart.Path)
	}

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to load chart: %v\n", err)
	}

	output, err := hc.template(ctx, chrt)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Helm templating error: %v\n", err)
		os.Exit(2)
	}

	// Print output to stdout
	fmt.Print(output)
}

func unmarshalHelmChart(crdPath string) (*HelmChart, error) {
	data, err := ioutil.ReadFile(crdPath)
	if err != nil {
		return nil, err
	}

	helmChart := &HelmChart{}

	err = yaml.Unmarshal(data, helmChart)
	if err != nil{
		return nil, err
	}

	return helmChart, nil
}

// pull loads a chart from a chart repository name version combination
func pull(ctx context.Context, repository string, name string, version string) (*chart.Chart, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return nil, nil
}

// clone loads a chart from a git repository path combination
func clone(ctx context.Context, repo string, ref string, path string) (*chart.Chart, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	_, err = git.PlainCloneContext(ctx, tmpDir, false, &git.CloneOptions{
		URL: repo,
		Progress: os.Stderr,
		Depth: 1,
		SingleBranch: true,
		ReferenceName: plumbing.NewBranchReferenceName(ref),
	})
	if err != nil {
		return nil, err
	}

	chart, err := loadAndUpdate(ctx, filepath.Join(tmpDir, path))
	if err != nil {
		return nil, err
	}

	return chart, nil
}

// template takes a loaded chart and returns `helm template .` as a string
func (c *HelmChart) template(ctx context.Context, ch *chart.Chart) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	client := newFakeClient(c.Name, c.Namespace)

	/* Values are merged in as follows:
		0. values from chart
		1. values from files
		2. values from inline map
		3. values from sops files
	*/
	mergedVals := ch.Values

	// Parse the values files
	for _, file := range c.ValueFiles {
		val, err := chartutil.ReadValuesFile(file)
		if err != nil {
			return "", err
		}

		mergedVals = mergeMaps(mergedVals, val)
	}

	// Parse map values
	mergedVals = mergeMaps(mergedVals, c.Values)

	// Parse any values files encrypted via sops
	for _, sopsFile := range c.SopsValueFiles {
		// Decrypt and read the values file (always a yaml for helm)
		decryptedData, err := decrypt.File(sopsFile, "yaml")
		if err != nil {
			return "", err
		}

		// Parse the decrypted data into helm values map
		val, err := chartutil.ReadValues(decryptedData)

		mergedVals = mergeMaps(mergedVals, val)
	}

	// Load in merged vals
	finalVals, err := chartutil.CoalesceValues(ch, mergedVals)
	if err != nil {
		return "", err
	}

	r, err := client.Run(ch, finalVals)
	if err != nil {
		return "", err
	}

	rendered := r.Manifest
	for _, hook := range r.Hooks {
		rendered = strings.Join([]string{rendered, hook.Manifest}, "\n---\n")
	}

	return rendered, nil
}

// newFakeClient returns a helm client solely used for templating with no intention of talking to any clusters
func newFakeClient(name string, namespace string) *action.Install {
	mem := driver.NewMemory()
	mem.SetNamespace(namespace)

	// Mock out enough configuration for templating
	cfg := &action.Configuration{
		Capabilities: chartutil.DefaultCapabilities,
		KubeClient: &fake.PrintingKubeClient{Out: ioutil.Discard},
		Releases: storage.Init(mem),
	}

	client := action.NewInstall(cfg)
	client.ReleaseName = name
	client.Namespace = namespace
	client.ClientOnly = true
	client.IncludeCRDs = true
	client.DisableHooks = false

	return client
}

func loadAndUpdate(ctx context.Context, path string) (*chart.Chart, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Load in the chart
	c, err := loader.Load(path)
	if err != nil {
		return nil, err
	}

	if err := action.CheckDependencies(c, c.Metadata.Dependencies); err != nil {
		//	Dependencies not found, out of date, or invalid checksum, pull them down
		settings := cli.New()
		mgr := downloader.Manager{
			Out: os.Stderr,
			ChartPath: path,
			Getters: getter.All(settings),
			RepositoryConfig: settings.RepositoryConfig,
			RepositoryCache: settings.RepositoryCache,
			SkipUpdate: true,
		}

		// This will rebuild _all_ dependencies
		// TODO: Rebuild only dependencies missing
		err = mgr.Update()
		if err != nil {
			return nil, err
		}

		// Reload chart when dependencies are rebuilt
		c, err = loader.Load(path)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

// mergeMaps will take in 2 maps and merge the two
func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

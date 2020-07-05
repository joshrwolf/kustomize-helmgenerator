package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/pkg/errors"
	"go.mozilla.org/sops/v3/decrypt"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
)

const apiVersion = "p1.dsop.io/v1beta1"
const kind = "HelmGenerator"

type kvMap map[string]string

type TypeMeta struct {
	APIVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
}

type ObjectMeta struct {
	Name        string `json:"name" yaml:"name"`
	Namespace   string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
	Labels      kvMap  `json:"labels,omitempty" yaml:"labels,omitempty"`
	Annotations kvMap  `json:"annotations,omitempty" yaml:"annotations,omitempty"`
}

type HelmGenerator struct {
	TypeMeta   `json:",inline" yaml:",inline"`
	ObjectMeta `json:"metadata" yaml:"metadata"`

	// ReleaseName is required
	ReleaseName string `json:"releaseName" yaml:"releaseName"`

	// ChartPath is required
	ChartPath string `json:"chartPath" yaml:"chartPath"`

	// Namespace is optional and will default to no namespace
	Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`

	// List of value files to use, will be used in order
	ValueFiles []string `json:"valueFiles,omitempty" yaml:"valueFiles,omitempty"`

	// Values is a generic map of values that are applied via --set
	Values string `json:"values,omitempty" yaml:"values,omitempty"`

	// SopsValueFiles is a generic list of values files encrypted via sops
	SopsValueFiles []string `json:"sopsValueFiles,omitempty" yaml:"sopsValueFiles,omitempty"`
}

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: HelmGenerator FILE")
		os.Exit(1)
	}

	out, err := processHelmGenerator(os.Args[1])
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
	}
	fmt.Print(out)
}

func processHelmGenerator(fn string) (string, error) {
	data, err := ioutil.ReadFile(fn)
	if err != nil {
	}

	input := HelmGenerator{
		TypeMeta: TypeMeta{},
		ObjectMeta: ObjectMeta{
			Annotations: make(kvMap),
		},
	}
	err = yaml.Unmarshal(data, &input)
	if err != nil {
	}

	templatedOut, err := input.helmTemplate()
	if err != nil {
	}

	return templatedOut, err
}

func (g *HelmGenerator) helmTemplate() (string, error) {
	settings := cli.New()

	// Go through the initialization formality even though we're just using this as a templating engine
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		log.Printf("%+v", err)
		return "", errors.Wrapf(err, "Error initializing helm client")
	}

	// Initialize client only dry run
	client := action.NewInstall(actionConfig)
	client.ClientOnly = true
	client.DryRun = true
	client.DisableHooks = false
	client.IncludeCRDs = true
	client.SubNotes = true

	// Load chart
	chart, err := loader.Load(g.ChartPath)
	if err != nil {
		return "", errors.Errorf("Failed to load chart from: %s", g.ChartPath)
	}

	// Parse spec
	client.ReleaseName = g.ReleaseName
	client.Namespace = g.Namespace

	// Start with default chart values
	vals, err := chartutil.CoalesceValues(chart, chart.Values)

	// Parse the values files
	for _, file := range g.ValueFiles {
		val, err := chartutil.ReadValuesFile(file)
		if err != nil {
			return "", errors.Errorf("Failed to read values from file: %s, %v", file, err)
		}

		vals = mergeMaps(vals, val)
	}

	// Parse map values
	val, err := chartutil.ReadValues([]byte(g.Values))
	if err != nil {
		return "", errors.Errorf("Failed to read byte values: %v", err)
	}
	vals = mergeMaps(vals, val)

	// Parse any values files encrypted via sops
	for _, sopsFile := range g.SopsValueFiles {
		// Decrypt and read the values file (always a yaml for helm)
		decryptedData, err := decrypt.File(sopsFile, "yaml")
		if err != nil {
			return "", errors.Errorf("Failed to read sops encrypted file: %s, %v", sopsFile, err)
		}

		// Parse the decrypted data into helm values map
		val, err := chartutil.ReadValues(decryptedData)

		vals = mergeMaps(vals, val)
	}

	// Run template (dry run install)
	r, err := client.Run(chart, vals)
	if err != nil {
		return "", errors.Errorf("Failed to run chart install: %v", err)
	}

	// Get base template
	templateString := r.Manifest

	// Cat base manifests with hook manifests
	for _, hook := range r.Hooks {
		templateString = strings.Join([]string{templateString, hook.Manifest}, "\n---\n")
	}

	return templateString, nil
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

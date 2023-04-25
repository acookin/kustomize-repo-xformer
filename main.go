package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	goyaml "gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type ImageRegistryTransformer struct {
	Registry    string `yaml:"registry" json:"registry"`
	NewRegistry string `yaml:"newRegistry" json:"newRegistry"`
}

type containerInfo struct {
	Containers     []*v1.Container
	ContainersPath []string
}

func getContainerInfo(item *yaml.RNode) (*containerInfo, error) {
	info := &containerInfo{
		Containers:     []*v1.Container{},
		ContainersPath: []string{},
	}
	for _, containerPath := range yaml.ConventionalContainerPaths {
		containers, err := item.Pipe(yaml.Lookup(containerPath...))
		if err == nil {
			containers.Document().Decode(&info.Containers)
			info.ContainersPath = containerPath
			return info, nil
		}
	}
	return nil, errors.New("missing no container info")
}

func cleanRegistryPath(path string) string {
	return filepath.Clean(path) + string(os.PathSeparator)
}

func main() {
	config := new(ImageRegistryTransformer)
	fn := func(items []*yaml.RNode) ([]*yaml.RNode, error) {
		for i, item := range items {
			meta, err := item.GetMeta()
			if err != nil {
				return nil, err
			}
			newRegistry := cleanRegistryPath(config.NewRegistry)
			oldRegistry := cleanRegistryPath(config.Registry)
			containerTypes := map[string]bool{
				"Pod":         true,
				"Deployment":  true,
				"CronJob":     true,
				"DaemonSet":   true,
				"StatefulSet": true,
			}
			if _, ok := containerTypes[meta.TypeMeta.Kind]; !ok {
				continue
			}
			containerInfo, err := getContainerInfo(item)
			if err != nil {
				return nil, err
			}
			for _, container := range containerInfo.Containers {
				if strings.HasPrefix(container.Image, oldRegistry) {
					container.Image = strings.Replace(container.Image, oldRegistry, newRegistry, 1)
				}
			}
			fmt.Fprintf(os.Stderr, "%+v\n", containerInfo.Containers)
			data, err := goyaml.Marshal(containerInfo.Containers)
			if err != nil {
				return nil, err
			}
			containersRnode, err := yaml.Parse(string(data))
			if err != nil {
				return nil, err
			}
			containersPath := containerInfo.ContainersPath[0 : len(containerInfo.ContainersPath)-1]
			items[i].PipeE(
				yaml.Lookup(containersPath...),
				yaml.SetField("containers", containersRnode),
			)

		}
		return items, nil
	}
	p := framework.SimpleProcessor{Config: config, Filter: kio.FilterFunc(fn)}
	cmd := command.Build(p, command.StandaloneDisabled, false)
	command.AddGenerateDockerfile(cmd)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

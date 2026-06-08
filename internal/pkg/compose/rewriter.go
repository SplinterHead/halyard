package compose

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/SplinterHead/halyard/internal/pkg/docker"
	"gopkg.in/yaml.v3"
)

// VersionConfigs parses the compose file, checks configs against Swarm, and updates version suffixes.
func VersionConfigs(ctx context.Context, dockerCli *docker.Client, stackName string, tmpDir string, composePath string) error {
	fullPath := filepath.Join(tmpDir, composePath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}

	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil
	}
	mapNode := root.Content[0]

	configsNode := findKey(mapNode, "configs")
	if configsNode == nil {
		return nil
	}

	servicesNode := findKey(mapNode, "services")

	// Map of old config name to new versioned config name
	renames := make(map[string]string)

	for i := 0; i < len(configsNode.Content); i += 2 {
		configNameNode := configsNode.Content[i]
		configBodyNode := configsNode.Content[i+1]

		fileNode := findKey(configBodyNode, "file")
		if fileNode == nil {
			continue
		}

		localFilePath := filepath.Join(tmpDir, filepath.Dir(composePath), fileNode.Value)
		localData, err := os.ReadFile(localFilePath)
		if err != nil {
			return fmt.Errorf("failed to read config file %s: %w", localFilePath, err)
		}

		baseName := configNameNode.Value
		newVersion, err := getNextConfigVersion(ctx, dockerCli, stackName, baseName, localData)
		if err != nil {
			return err
		}

		if newVersion != baseName {
			renames[baseName] = newVersion
			configNameNode.Value = newVersion
		}
	}

	if len(renames) > 0 && servicesNode != nil {
		updateServiceConfigs(servicesNode, renames)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return err
	}

	return os.WriteFile(fullPath, buf.Bytes(), 0644)
}

func getNextConfigVersion(ctx context.Context, cli *docker.Client, stackName, baseName string, localData []byte) (string, error) {
	// target prefix: stackName_baseName or stackName_baseName-v
	prefix := fmt.Sprintf("%s_%s", stackName, baseName)
	args := filters.NewArgs()
	args.Add("name", prefix)
	
	configs, err := cli.ConfigList(ctx, types.ConfigListOptions{Filters: args})
	if err != nil {
		return "", err
	}

	var maxVersion int = 1
	var maxConfig swarm.Config
	var foundAny bool

	regex := regexp.MustCompile(fmt.Sprintf(`^%s(?:-v(\d+))?$`, regexp.QuoteMeta(prefix)))

	for _, c := range configs {
		matches := regex.FindStringSubmatch(c.Spec.Name)
		if matches != nil {
			foundAny = true
			if matches[1] != "" {
				v, _ := strconv.Atoi(matches[1])
				if v > maxVersion {
					maxVersion = v
					maxConfig = c
				}
			} else {
				if 1 >= maxVersion {
					maxVersion = 1
					maxConfig = c
				}
			}
		}
	}

	if !foundAny {
		// First time, just use baseName
		return baseName, nil
	}

	// We have a highest version config. Inspect its data.
	configData, _, err := cli.ConfigInspectWithRaw(ctx, maxConfig.ID)
	if err != nil {
		return "", err
	}

	if bytes.Equal(configData.Spec.Data, localData) {
		// Unchanged. Use the existing version name
		if maxVersion == 1 {
			return baseName, nil
		}
		return fmt.Sprintf("%s-v%d", baseName, maxVersion), nil
	}

	// Changed. Increment version.
	return fmt.Sprintf("%s-v%d", baseName, maxVersion+1), nil
}

func findKey(mapNode *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(mapNode.Content); i += 2 {
		if mapNode.Content[i].Value == key {
			return mapNode.Content[i+1]
		}
	}
	return nil
}

func updateServiceConfigs(servicesNode *yaml.Node, renames map[string]string) {
	for i := 1; i < len(servicesNode.Content); i += 2 {
		svcMap := servicesNode.Content[i]
		svcConfigs := findKey(svcMap, "configs")
		if svcConfigs == nil || svcConfigs.Kind != yaml.SequenceNode {
			continue
		}

		for _, item := range svcConfigs.Content {
			if item.Kind == yaml.ScalarNode {
				if newName, ok := renames[item.Value]; ok {
					item.Value = newName
				}
			} else if item.Kind == yaml.MappingNode {
				sourceNode := findKey(item, "source")
				if sourceNode != nil && sourceNode.Kind == yaml.ScalarNode {
					if newName, ok := renames[sourceNode.Value]; ok {
						sourceNode.Value = newName
					}
				}
			}
		}
	}
}

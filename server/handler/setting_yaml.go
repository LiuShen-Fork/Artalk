package handler

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/artalkjs/artalk/v2/internal/config"
	"github.com/artalkjs/artalk/v2/internal/core"
	"gopkg.in/yaml.v3"
)

var (
	errConfigFileUnavailable = errors.New("config file path is not available")
	errConfigRootNotMapping  = errors.New("config file root is not a mapping")
)

// patchYAMLValues 修改 YAML 文档中的若干标量 / 列表值，返回新的文件内容
//
// updates 的 key 为点分路径（例如 "security.rate_limit.ip.minute"），值可以是标量，
// 也可以是 []string。基于 yaml.Node 读写，尽量保留用户的注释与键顺序。
func patchYAMLValues(src []byte, updates map[string]any) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, err
	}

	root := yamlDocumentRoot(&doc)
	if root.Kind != yaml.MappingNode {
		return nil, errConfigRootNotMapping
	}

	paths := make([]string, 0, len(updates))
	for path := range updates {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		if err := setYAMLPath(root, strings.Split(path, "."), updates[path]); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}

	return yaml.Marshal(&doc)
}

func yamlDocumentRoot(doc *yaml.Node) *yaml.Node {
	if len(doc.Content) == 0 {
		root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{root}
	}
	return doc.Content[0]
}

func setYAMLPath(node *yaml.Node, parts []string, value any) error {
	if node.Kind != yaml.MappingNode {
		node.Kind = yaml.MappingNode
		node.Tag = "!!map"
		node.Value = ""
		node.Content = nil
	}
	if len(parts) == 0 {
		return errors.New("empty config path")
	}

	key := parts[0]
	if len(parts) == 1 {
		return setYAMLMappingValue(node, key, value)
	}

	child := yamlMappingValue(node, key)
	if child == nil {
		child = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		node.Content = append(node.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, child)
	}
	return setYAMLPath(child, parts[1:], value)
}

func setYAMLMappingValue(mapping *yaml.Node, key string, value any) error {
	var node yaml.Node
	if err := node.Encode(value); err != nil {
		return err
	}

	if existing := yamlMappingValue(mapping, key); existing != nil {
		*existing = node
		return nil
	}

	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, &node)
	return nil
}

func yamlMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// writeConfigFile 将 updates 写入配置文件并重新加载配置（不重启服务）
//
// 与 PUT /settings 的整体覆盖不同，这里只改动指定字段，用户的注释与其他配置保持原样。
func writeConfigFile(app *core.App, updates map[string]any) (*config.Config, error) {
	cfgFile := strings.TrimSpace(app.Conf().GetCfgFileLoaded())
	if cfgFile == "" {
		return nil, errConfigFileUnavailable
	}

	src, err := os.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	out, err := patchYAMLValues(src, updates)
	if err != nil {
		return nil, err
	}

	// 保留原文件权限，先写临时文件再重命名，避免中途失败留下半个配置文件
	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(cfgFile); statErr == nil {
		mode = info.Mode().Perm()
	}
	tmpFile := cfgFile + ".tmp"
	if err := os.WriteFile(tmpFile, out, mode); err != nil {
		return nil, err
	}
	if err := os.Rename(tmpFile, cfgFile); err != nil {
		_ = os.Remove(tmpFile)
		return nil, err
	}

	return config.NewFromFile(cfgFile)
}

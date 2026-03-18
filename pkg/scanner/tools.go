package scanner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	fileutil "github.com/projectdiscovery/utils/file"
	"gopkg.in/yaml.v3"
)

func GetToolDir() (string, error) {
	exe, _ := os.Executable()
	binDir := filepath.Dir(exe)
	if fileutil.FolderExists(filepath.Join(binDir, "tools")) {
		return filepath.Join(binDir, "tools"), nil
	}
	if fileutil.FolderExists("tools") {
		return "tools", nil
	}
	home, _ := os.UserHomeDir()
	confDir := filepath.Join(home, ".config", "zetgrep", "tools")
	if fileutil.FolderExists(confDir) {
		return confDir, nil
	}
	return "tools", nil
}

func LoadTools() []models.Tool {
	dir, _ := GetToolDir()
	return LoadToolsFrom(dir)
}

func LoadToolFromFile(f string) (models.Tool, error) {
	var t models.Tool
	b, err := os.ReadFile(f)
	if err != nil {
		return t, err
	}
	if err := yaml.Unmarshal(b, &t); err != nil {
		return t, err
	}
	if t.ID == "" {
		return t, fmt.Errorf("tool in %s has no ID", f)
	}
	if t.Field == "" {
		t.Field = t.Name
	}
	return t, nil
}

func LoadToolsFrom(dirs ...string) []models.Tool {
	var tools []models.Tool
	for _, dir := range dirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
		for _, f := range files {
			if t, err := LoadToolFromFile(f); err == nil {
				tools = append(tools, t)
			}
		}
	}
	return tools
}

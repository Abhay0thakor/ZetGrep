package scanner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Abhay0thakor/ZetGrep/pkg/models"
	fileutil "github.com/projectdiscovery/utils/file"
	"gopkg.in/yaml.v3"
)

func GetToolDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "tools", err
	}
	binDir := filepath.Dir(exe)
	if fileutil.FolderExists(filepath.Join(binDir, "tools")) {
		return filepath.Join(binDir, "tools"), nil
	}
	if fileutil.FolderExists("tools") {
		return "tools", nil
	}
	home, err := os.UserHomeDir()
	if err == nil {
		confDir := filepath.Join(home, ".config", "gf", "tools")
		if fileutil.FolderExists(confDir) {
			return confDir, nil
		}
	}
	return "tools", nil
}

func LoadTools() ([]models.Tool, error) {
	dir, err := GetToolDir()
	if err != nil {
		return nil, err
	}
	return LoadToolsFrom(dir)
}

func LoadToolFromFile(f string) (models.Tool, error) {
	var t models.Tool
	b, err := os.ReadFile(f)
	if err != nil {
		return t, fmt.Errorf("error reading tool file %s: %w", f, err)
	}
	if err := yaml.Unmarshal(b, &t); err != nil {
		return t, fmt.Errorf("error parsing tool YAML %s: %w", f, err)
	}
	if t.ID == "" {
		return t, fmt.Errorf("tool in %s has no ID", f)
	}
	if t.Field == "" {
		t.Field = t.Name
	}
	return t, nil
}

func LoadToolsFrom(dirs ...string) ([]models.Tool, error) {
	var tools []models.Tool
	var errs []string
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		files, err := filepath.Glob(filepath.Join(dir, "*.yaml"))
		if err != nil {
			errs = append(errs, fmt.Sprintf("error listing tools in %s: %v", dir, err))
			continue
		}
		for _, f := range files {
			if t, err := LoadToolFromFile(f); err == nil {
				tools = append(tools, t)
			} else {
				errs = append(errs, err.Error())
			}
		}
	}
	var err error
	if len(errs) > 0 {
		err = errors.New(strings.Join(errs, "; "))
	}
	return tools, err
}

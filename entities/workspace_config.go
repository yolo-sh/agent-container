package entities

import (
	"encoding/json"
	"os"
)

type WorkspaceConfig struct {
	Repositories []WorkspaceConfigRepository `json:"repositories"`
}

type WorkspaceConfigRepository struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	RootDirPath string `json:"root_dir_path"`
	IsMainRepo  bool   `json:"is_main_repo"`
}

func NewWorkspaceConfig() *WorkspaceConfig {
	return &WorkspaceConfig{
		Repositories: []WorkspaceConfigRepository{},
	}
}

func LoadWorkspaceConfig(
	workspaceConfigFilePath string,
) (*WorkspaceConfig, error) {

	workspaceConfigFileContent, err := os.ReadFile(workspaceConfigFilePath)

	if err != nil {
		return nil, err
	}

	var workspaceConfig *WorkspaceConfig
	err = json.Unmarshal(workspaceConfigFileContent, &workspaceConfig)

	if err != nil {
		return nil, err
	}

	return workspaceConfig, nil
}

func SaveWorkspaceConfigAsFile(
	workspaceConfigFilePath string,
	workspaceConfig *WorkspaceConfig,
) error {

	workspaceConfigAsJSON, err := json.Marshal(workspaceConfig)

	if err != nil {
		return err
	}

	err = os.WriteFile(
		workspaceConfigFilePath,
		workspaceConfigAsJSON,
		os.FileMode(0660),
	)

	if err != nil {
		return err
	}

	// Overwrite umask.
	// See: https://stackoverflow.com/questions/50257981/ioutils-writefile-not-respecting-permissions
	return os.Chmod(
		workspaceConfigFilePath,
		0660,
	)
}

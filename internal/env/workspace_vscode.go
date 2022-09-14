package env

import (
	"encoding/json"
	"os"
)

var LanguagesVSCodeExtensions = map[string][]string{
	"Go":         {"golang.go"},
	"Ruby":       {"rebornix.Ruby"},
	"Rust":       {"rust-lang.rust-analyzer"},
	"Python":     {"ms-python.python"},
	"Java":       {"vscjava.vscode-java-pack"},
	"C":          {"ms-vscode.cpptools-extension-pack"},
	"C++":        {"ms-vscode.cpptools-extension-pack"},
	"CMake":      {"ms-vscode.cpptools-extension-pack"},
	"Dockerfile": {"ms-azuretools.vscode-docker"},
}

// VSCodeWorkspaceConfig matches .code-workspace schema.
// See: https://code.visualstudio.com/docs/editor/multi-root-workspaces#_workspace-file-schema
type VSCodeWorkspaceConfig struct {
	Folders    []VSCodeWorkspaceConfigFolder   `json:"folders"`
	Settings   map[string]interface{}          `json:"settings"`
	Extensions VSCodeWorkspaceConfigExtensions `json:"extensions"`
}

type VSCodeWorkspaceConfigFolder struct {
	Path string `json:"path"`
}

type VSCodeWorkspaceConfigExtensions struct {
	Recommendations []string `json:"recommendations"`
}

func buildInitialVSCodeWorkspaceConfig(
	languagesUsed []string,
) VSCodeWorkspaceConfig {

	return VSCodeWorkspaceConfig{
		Folders: []VSCodeWorkspaceConfigFolder{},
		Settings: map[string]interface{}{
			"remote.autoForwardPorts":      true,
			"remote.restoreForwardedPorts": true,
			// Auto-detect (using "/proc") and forward opened port.
			// Way better than "output" that parse terminal output.
			// See: https://github.com/microsoft/vscode/issues/143958#issuecomment-1050959241
			"remote.autoForwardPortsSource": "process",
			// We overwrite the $PATH environment variable in integrated terminal
			// because RVM displays warnings when VSCode changes the order of the paths.
			// See: https://github.com/microsoft/vscode/issues/70248
			"terminal.integrated.env.linux": map[string]interface{}{
				"PATH": "${env:PATH}",
			},
			// Socket is not supported by the container agent.
			"remote.SSH.remoteServerListenOnSocket": false,
			"remote.downloadExtensionsLocally":      false,
			"window.title":                          "${activeEditorShort}${separator}${remoteName} (via Yolo)",
		},
		Extensions: VSCodeWorkspaceConfigExtensions{
			Recommendations: convertLanguagesToVSCodeExtensions(languagesUsed),
		},
	}
}

func saveVSCodeWorkspaceConfigAsFile(
	vscodeWorkspaceConfigFilePath string,
	vscodeWorkspaceConfig VSCodeWorkspaceConfig,
) error {

	vscodeWorkspaceConfigAsJSON, err := json.Marshal(&vscodeWorkspaceConfig)

	if err != nil {
		return err
	}

	return os.WriteFile(
		vscodeWorkspaceConfigFilePath,
		vscodeWorkspaceConfigAsJSON,
		os.FileMode(0644),
	)
}

func convertLanguagesToVSCodeExtensions(
	languages []string,
) []string {

	vscodeExts := []string{}

	for _, language := range languages {
		if languageExtensions, ok := LanguagesVSCodeExtensions[language]; ok {
			vscodeExts = mergeVSCodeExtensionsRecos(
				vscodeExts,
				languageExtensions,
			)
		}
	}

	return vscodeExts
}

func mergeVSCodeExtensionsRecos(
	currentRecos []string,
	recosToAdd []string,
) []string {

	mergedRecos := []string{}
	hasRecoMap := map[string]bool{}

	for _, currentReco := range currentRecos {
		mergedRecos = append(mergedRecos, currentReco)
		hasRecoMap[currentReco] = true
	}

	for _, recoToAdd := range recosToAdd {
		_, alreadyHasReco := hasRecoMap[recoToAdd]

		if alreadyHasReco {
			continue
		}

		mergedRecos = append(mergedRecos, recoToAdd)
		hasRecoMap[recoToAdd] = true
	}

	return mergedRecos
}

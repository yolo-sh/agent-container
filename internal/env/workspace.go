package env

import (
	"path/filepath"

	"github.com/yolo-sh/agent-container/constants"
	"github.com/yolo-sh/agent-container/entities"
	"github.com/yolo-sh/agent-container/internal/system"
)

func PrepareWorkspace(
	workspaceConfig *entities.WorkspaceConfig,
	repoOwner string,
	repoName string,
	languagesUsedInRepo []string,
) error {
	vscodeWorkspaceConfig := buildInitialVSCodeWorkspaceConfig(languagesUsedInRepo)

	// The method "PrepareWorkspace" could
	// be called multiple times in case of error
	// so we need to make sure that our code is idempotent
	err := system.NewFileManager().RemoveDirContent(
		constants.WorkspaceDirPath,
	)

	if err != nil {
		return err
	}

	err = addRepoToWorkspace(
		repoOwner,
		repoName,
		workspaceConfig,
		&vscodeWorkspaceConfig,
	)

	if err != nil {
		return err
	}

	err = saveVSCodeWorkspaceConfigAsFile(
		constants.VSCodeWorkspaceConfigFilePath,
		vscodeWorkspaceConfig,
	)

	if err != nil {
		return err
	}

	return entities.SaveWorkspaceConfigAsFile(
		constants.WorkspaceConfigFilePath,
		workspaceConfig,
	)
}

func addRepoToWorkspace(
	repoOwner string,
	repoName string,
	workspaceConfig *entities.WorkspaceConfig,
	vscodeWorkspaceConfig *VSCodeWorkspaceConfig,
) error {

	repoDirPathInWorkspace := filepath.Join(
		constants.WorkspaceDirPath,
		repoName,
	)

	err := cloneGitHubRepo(
		repoOwner,
		repoName,
		repoDirPathInWorkspace,
	)

	if err != nil {
		return err
	}

	workspaceConfigRepository := entities.WorkspaceConfigRepository{
		Owner:       repoOwner,
		Name:        repoName,
		RootDirPath: repoDirPathInWorkspace,
		IsMainRepo:  true,
	}

	workspaceConfig.Repositories = append(
		workspaceConfig.Repositories,
		workspaceConfigRepository,
	)

	vscodeWorkspaceConfig.Folders = append(
		vscodeWorkspaceConfig.Folders,
		VSCodeWorkspaceConfigFolder{
			Path: repoDirPathInWorkspace,
		},
	)

	return nil
}

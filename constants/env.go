package constants

const (
	YoloUserName        = "yolo"
	YoloUserHomeDirPath = "/home/" + YoloUserName

	YoloConfigDirPath = "/yolo-config"

	DockerImageTag  = "0.0.3"
	DockerImageName = "ghcr.io/yolo-sh/workspace-full:" + DockerImageTag

	DockerContainerName      = "yolo-env-container"
	DockerContainerIPAddress = "172.20.0.2"

	WorkspaceDirPath = YoloUserHomeDirPath + "/workspace"

	WorkspaceConfigDirPath  = YoloConfigDirPath + "/workspace"
	WorkspaceConfigFilePath = WorkspaceConfigDirPath + "/config.json"

	VSCodeWorkspaceConfigFilePath = WorkspaceConfigDirPath + "/default.code-workspace"

	GitHubPublicSSHKeyFilePath = YoloUserHomeDirPath + "/.ssh/" + YoloUserName + "-github.pub"
	GitHubPublicGPGKeyFilePath = YoloUserHomeDirPath + "/.gnupg/" + YoloUserName + "-github-gpg-public.pgp"
)

var DockerContainerEntrypoint = []string{
	"/sbin/init",
	"--log-level=err",
}

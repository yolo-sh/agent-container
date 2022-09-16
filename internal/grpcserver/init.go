package grpcserver

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/yolo-sh/agent-container/constants"
	"github.com/yolo-sh/agent-container/entities"
	"github.com/yolo-sh/agent-container/internal/env"
	"github.com/yolo-sh/agent-container/proto"
)

//go:embed init.sh
var initScript string

func (*agentServer) Init(
	req *proto.InitRequest,
	stream proto.Agent_InitServer,
) error {

	err := stream.Send(&proto.InitReply{
		LogLineHeader: fmt.Sprintf(
			"Executing %s",
			constants.InitScriptRepoPath,
		),
	})

	if err != nil {
		return err
	}

	initScriptFilePath, err := createInitScriptFile()

	if err != nil {
		return err
	}

	defer os.Remove(initScriptFilePath)

	initCmd := buildInitCmd(initScriptFilePath, req)

	stdoutReader, err := buildInitCmdStdoutReader(initCmd)

	if err != nil {
		return err
	}

	stderrReader, err := buildInitCmdStderrReader(initCmd)

	if err != nil {
		return err
	}

	if err := initCmd.Start(); err != nil {
		return err
	}

	stdoutHandlerChan := make(chan error, 1)

	go func() {
		stdoutHandlerChan <- handleInitCmdOutput(
			stdoutReader,
			stream,
		)
	}()

	stderrHandlerChan := make(chan error, 1)

	go func() {
		stderrHandlerChan <- handleInitCmdOutput(
			stderrReader,
			stream,
		)
	}()

	stdoutHandlerErr := <-stdoutHandlerChan
	stderrHandlerErr := <-stderrHandlerChan

	if stdoutHandlerErr != nil {
		return stdoutHandlerErr
	}

	if stderrHandlerErr != nil {
		return stderrHandlerErr
	}

	// It is incorrect to call Wait
	// before all reads from the pipes have completed.
	// See StderrPipe() / StdoutPipe() documentation.
	if err := initCmd.Wait(); err != nil {
		return err
	}

	githubSSHPublicKeyContent, err := readGitHubSSHPublicKey(
		constants.GitHubPublicSSHKeyFilePath,
	)

	if err != nil {
		return err
	}

	githubGPGPublicKeyContent, err := readGitHubGPGPublicKey(
		constants.GitHubPublicGPGKeyFilePath,
	)

	if err != nil {
		return err
	}

	err = stream.Send(&proto.InitReply{
		GithubSshPublicKeyContent: &githubSSHPublicKeyContent,
		GithubGpgPublicKeyContent: &githubGPGPublicKeyContent,
	})

	if err != nil {
		return err
	}

	workspaceConfig := entities.NewWorkspaceConfig()

	return env.PrepareWorkspace(
		workspaceConfig,
		req.EnvRepoOwner,
		req.EnvRepoName,
		req.EnvRepoLanguagesUsed,
	)
}

func createInitScriptFile() (string, error) {
	initScriptFile, err := os.CreateTemp("", "yolo-init-script-*")

	if err != nil {
		return "", err
	}

	err = fillInitScriptFile(initScriptFile)

	if err != nil {
		return "", err
	}

	// Opened file cannot be executed at the same time.
	// Prevent "fork/exec text file busy" error.
	err = closeInitScriptFile(initScriptFile)

	if err != nil {
		return "", err
	}

	err = addExecPermsToInitScriptFile(initScriptFile)

	if err != nil {
		return "", err
	}

	return initScriptFile.Name(), nil
}

func fillInitScriptFile(initScriptFile *os.File) error {
	_, err := initScriptFile.WriteString(initScript)
	return err
}

func closeInitScriptFile(initScriptFile *os.File) error {
	return initScriptFile.Close()
}

func addExecPermsToInitScriptFile(initScriptFile *os.File) error {
	return os.Chmod(
		initScriptFile.Name(),
		os.FileMode(0700),
	)
}

func buildInitCmd(
	initScriptFilePath string,
	req *proto.InitRequest,
) *exec.Cmd {

	initCmd := exec.Command(initScriptFilePath)

	initCmd.Dir = path.Dir(initScriptFilePath)
	initCmd.Env = buildInitCmdEnvVars(req)

	return initCmd
}

func buildInitCmdEnvVars(req *proto.InitRequest) []string {
	return []string{
		fmt.Sprintf("GITHUB_USER_EMAIL=%s", req.GithubUserEmail),
		fmt.Sprintf("USER_FULL_NAME=%s", req.UserFullName),
	}
}

func buildInitCmdStderrReader(initCmd *exec.Cmd) (*bufio.Reader, error) {
	stderrPipe, err := initCmd.StderrPipe()

	if err != nil {
		return nil, err
	}

	return bufio.NewReader(stderrPipe), nil
}

func buildInitCmdStdoutReader(initCmd *exec.Cmd) (*bufio.Reader, error) {
	stdoutPipe, err := initCmd.StdoutPipe()

	if err != nil {
		return nil, err
	}

	return bufio.NewReader(stdoutPipe), nil
}

func handleInitCmdOutput(
	outputReader *bufio.Reader,
	stream proto.Agent_InitServer,
) error {

	for {
		outputLine, err := outputReader.ReadString('\n')

		if err != nil && errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}

		err = stream.Send(&proto.InitReply{
			LogLine: outputLine,
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func readGitHubSSHPublicKey(sshPublicKeyFilePath string) (string, error) {
	sshPublicKeyContent, err := os.ReadFile(sshPublicKeyFilePath)

	if err != nil {
		return "", err
	}

	return string(sshPublicKeyContent), nil
}

func readGitHubGPGPublicKey(gpgPublicKeyFilePath string) (string, error) {
	gpgPublicKeyContent, err := os.ReadFile(gpgPublicKeyFilePath)

	if err != nil {
		return "", err
	}

	return string(gpgPublicKeyContent), nil
}

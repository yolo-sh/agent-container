package constants

const (
	GRPCServerAddrProtocol = "unix"
	GRPCServerAddr         = YoloConfigDirPath + "/agent-container-grpc.sock"
	GRPCServerUri          = "unix://" + GRPCServerAddr

	InitScriptRepoPath = "yolo-sh/agent-container/internal/grpcserver/init.sh"
)

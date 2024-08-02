package sync

// SyncSources makes a list of all supported sync sources globally available. 
var SyncSources = map[string]ComputeUserFunc{
	"authentik": ComputeAuthentikUsers,
}
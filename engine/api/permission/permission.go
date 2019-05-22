package permission

const (
	// PermissionRead  read permission on the resource
	PermissionRead = 4
	// PermissionReadExecute  read & execute permission on the resource
	PermissionReadExecute = 5
	// PermissionReadWriteExecute read/execute/write permission on the resource
	PermissionReadWriteExecute = 7
)

var (
	// SharedInfraGroupID must be init from elsewhere with group.SharedInfraGroup
	SharedInfraGroupID int64

	// DefaultGroupID same as SharedInfraGroupID
	DefaultGroupID int64
)

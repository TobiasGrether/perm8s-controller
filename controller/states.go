package controller

const controllerAgentName = "perm8s-controller"

const (
    SuccessSynced  = "Synced"
    SuccessCreated = "Created"
    ErrResourceExists = "ErrResourceExists"
    MessageResourceExists = "Resource %q already exists and is not managed by User"
    MessageUserSynced  = "User synced successfully"
    MessageUserCreated = "User created successfully"
    MessageGroupSynced = "Group synced successfully"
    FieldManager = controllerAgentName
)
package zitadel

// Role representa un rol de proyecto en Zitadel.
type Role string

const (
	RoleCollaborator Role = "collaborator"
	RoleInvited      Role = "invited"
	RolePlatAdmin    Role = "plat-admin"
	RoleSysAdmin     Role = "sys-admin"
)
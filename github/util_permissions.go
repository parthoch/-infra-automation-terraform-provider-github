package github

const (
	pullPermission     string = "pull"
	triagePermission   string = "triage"
	pushPermission     string = "push"
	maintainPermission string = "maintain"
	adminPermission    string = "admin"
	writePermission    string = "write"
	readPermission     string = "read"
)

func getInvitationPermission(permission string) string {
	// Permissions for some GitHub API routes are expressed as "read",
	// "write", and "admin"; in other places, they are expressed as "pull",
	// "push", and "admin".
	if permission == readPermission {
		return pullPermission
	} else if permission == writePermission {
		return pushPermission
	}
	return permission
}

package borm

type RoleName string
type RoleType string
type Role struct {
	RoleMethods

	Name RoleName
	Type RoleType
}
type RoleMethods interface {
	GetName() RoleName
	GetType() RoleType
}

const (
	USER RoleType = "user"
)

func (r *Role) GetName() RoleName {
	return r.Name
}
func (r *Role) GetType() RoleType {
	return r.Type
}

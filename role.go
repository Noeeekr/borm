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
type EnumMethods interface {
	GetValues() []string
}

type Enum struct {
	EnumMethods

	*Role
	options []string
}

const (
	ENUM RoleType = "enum"
	USER RoleType = "user"
)

func (r *Role) GetName() RoleName {
	return r.Name
}
func (r *Role) GetType() RoleType {
	return r.Type
}
func (r *Enum) GetValues() []string {
	return r.options
}

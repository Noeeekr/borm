package registers

import "strings"

type RoleName string
type RoleType string
type RoleCache map[RoleName]RoleMethods
type Role struct {
	RoleMethods

	Name RoleName
	Type RoleType
}
type RoleMethods interface {
	RoleName() RoleName
	RoleType() RoleType
}
type EnumMethods interface {
	Values() []string
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

func (r *Role) RoleName() RoleName {
	return r.Name
}
func (r *Role) RoleType() RoleType {
	return r.Type
}
func (r *Enum) Values() []string {
	return r.options
}
func (r *RoleCache) Enum(name string, values ...string) *Enum {
	roleName := RoleName(strings.ToLower(name))
	enum := &Enum{Role: &Role{Name: roleName, Type: ENUM}, options: values}
	(*r)[roleName] = enum
	return enum
}
func NewRoleCache() *RoleCache {
	return &RoleCache{}
}

package registers

import (
	"strings"

	"github.com/Noeeekr/borm/common"
)

type UserMethods interface {
	PrivilegedTables() []*Table
	GrantPrivileges(*Table, ...TablePrivilege) *UserPrivilegeRequest
	ToColumns(...TableColumnName) *User
}
type User struct {
	UserMethods

	*Role
}
type UserPrivilegeRequest struct {
	*User
	table      *Table
	privileges []TablePrivilege
	columns    []TableColumnName
}

func (r *User) GrantPrivileges(t *Table, p ...TablePrivilege) *UserPrivilegeRequest {
	if len(p) == 0 {
		t.Error = common.NewError().Description("User privileges should not be empty.").Status(common.ErrEmpty)
	}
	return &UserPrivilegeRequest{
		User:       r,
		table:      t,
		privileges: p,
		columns:    []TableColumnName{},
	}
}
func (r *UserPrivilegeRequest) ToColumns(c ...TableColumnName) *User {
	if len(c) == 0 {
		for _, fieldName := range r.table.Fields {
			r.columns = append(r.columns, fieldName.Name)
		}
	}
	r.columns = append(r.columns, c...)
	return r.User
}
func (r *RoleCache) User(name string) *User {
	roleName := RoleName(strings.ReplaceAll(strings.ToLower(name), " ", "_"))
	user := &User{Role: &Role{Name: roleName, Type: USER}}
	r.cache[name] = user
	return user
}

package registers

import (
	"strings"
)

type UserMethods interface {
	// PrivilegedTables() []*Table
	// GrantPrivileges(*Table, ...TablePrivilege) *UserPrivilegeRequest
	ToColumns(...TableColumnName) *User
	Password() string
}
type User struct {
	password string
	UserMethods

	*Role
}
type UserPrivilegeRequest struct {
	*User
	table *Table
	// privileges []TablePrivilege
	columns []TableColumnName
}

func (u *User) Password() string {
	return u.password
}

// func (u *User) GrantPrivileges(t *Table, p ...TablePrivilege) *UserPrivilegeRequest {
// 	if len(p) == 0 {
// 		t.Error = common.NewError("User privileges should not be empty.").Status(common.ErrEmpty)
// 	}
// 	return &UserPrivilegeRequest{
// 		User:       u,
// 		table:      t,
// 		privileges: p,
// 		columns:    []TableColumnName{},
// 	}
// }

// If empty adds all columns to grant privilege
func (r *UserPrivilegeRequest) ToColumns(c ...TableColumnName) *User {
	if len(c) == 0 {
		for _, fieldName := range r.table.Fields {
			r.columns = append(r.columns, fieldName.Name)
		}
	}
	r.columns = append(r.columns, c...)
	return r.User
}
func (r *User) WithPassword(password string) *User {
	r.password = password
	return r
}
func (r *RoleCache) User(name, password string) *User {
	user := NewUser(name, password)
	(*r)[user.Name] = user
	return user
}

func NewUser(name, password string) *User {
	roleName := RoleName(strings.ReplaceAll(strings.ToLower(name), " ", "_"))
	return &User{Role: &Role{Name: roleName, Type: USER}, password: password}
}

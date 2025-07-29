package borm

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

//	type UserPrivilegeRequest struct {
//		*User
//		table *TableRegistor
//		// privileges []TablePrivilege
//		columns []TableColumnName
//	}
//
//	func (u *User) GrantPrivileges(t *Table, p ...TablePrivilege) *UserPrivilegeRequest {
//		if len(p) == 0 {
//			tError = common.NeError("User privileges should not be empty.").Status(commonErrEmpty)
//		}
//		return &UserPrivilegeRequest{
//			User:       u,
//			table:      t,
//			privileges: p,
//			columns:    []TableColumnName{},
//		}
//	}
//
// If empty adds all columns to grant privilege
//
//	func (r *UserPrivilegeRequest) ToColumns(c ...TableColumnName) *User {
//		if len(c) == 0 {
//			for _, fieldName := range r.table.Fields {
//				r.columns = append(r.columns, fieldName.Name)
//			}
//		}
//		r.columns = append(r.columns, c...)
//		return r.User
//	}
func (u *User) Password() string {
	return u.password
}
func (r *User) WithPassword(password string) *User {
	r.password = password
	return r
}
func RegisterUser(name, password string) *User {
	user := newUser(name, password)
	(*roles)[user.Name] = user
	return user
}
func newUser(name, password string) *User {
	roleName := RoleName(strings.ReplaceAll(strings.ToLower(name), " ", "_"))
	return &User{Role: &Role{Name: roleName, Type: USER}, password: password}
}

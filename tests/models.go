package main

import "time"

type UserRole string

const (
	STUDENT UserRole = "STUDENT"
	TEACHER UserRole = "TEACHER"
	ADMIN   UserRole = "ADMIN"
)

type Id struct {
	Id int `borm:"(TYPE, SERIAL) (CONSTRAINTS, PRIMARY KEY)"`
}
type Users struct {
	*Id

	Name     string    `borm:"(CONSTRAINTS, NOT NULL)"`
	Email    string    `borm:"(CONSTRAINTS, NOT NULL, UNIQUE)"`
	Password string    `borm:"(CONSTRAINTS, NOT NULL)"`
	Role     *UserRole `borm:"(TYPE, user_role)"`

	SpecificWordA string `borm:"(NAME, specific_a)"`
	SpecificWordB string `borm:"(NAME, specific_b)"`
	SpecificWordC string `borm:"(NAME, specific_c)"`

	DeletedAt time.Time `borm:"(NAME, deleted_at)"`
	UpdatedAt time.Time `borm:"(NAME, updated_at)"`
	CreatedAt time.Time `borm:"(NAME, created_at)"`
}
type Notifications struct {
	Id          int    `borm:"(TYPE, SERIAL) (CONSTRAINTS, PRIMARY KEY)"`
	IssuerId    int    `borm:"(NAME, issuer_id) (FOREIGN KEY, USERS, ID)"`
	Title       string `borm:"(CONSTRAINTS, DEFAULT 'empty title')"`
	Description string

	ThisFieldShouldNotExist int `borm:"(IGNORE)"`
}
type UsersNotifications struct {
	UserId         int `borm:"(NAME, user_id) (FOREIGN KEY, USERS, ID)"`
	NotificationId int `borm:"(NAME, notification_id) (FOREIGN KEY, NOTIFICATIONS, ID)"`
}
type FilterClassOptions struct {
	A string
	B string
	C string
	D string
	E UserRole
	F *time.Time
}

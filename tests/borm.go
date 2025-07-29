package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Noeeekr/borm"
)

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
	Id

	Name     string   `borm:"(CONSTRAINTS, NOT NULL)"`
	Email    string   `borm:"(CONSTRAINTS, NOT NULL, UNIQUE)"`
	Password string   `borm:"(CONSTRAINTS, NOT NULL)"`
	Role     UserRole `borm:"(CONSTRAINTS, NOT NULL) (TYPE, user_role)"`

	DeletedAt time.Time `borm:"(NAME, deleted_at)"`
	UpdatedAt time.Time `borm:"(NAME, updated_at)"`
	CreatedAt time.Time `borm:"(NAME, created_at)"`
}
type Notifications struct {
	Id          int `borm:"(TYPE, SERIAL) (CONSTRAINTS, PRIMARY KEY)"`
	IssuerId    int `borm:"(NAME, issuer_id) (FOREIGN KEY, USERS, ID)"`
	Title       string
	Description string
}
type UsersNotifications struct {
	UserId         int `borm:"(NAME, user_id) (FOREIGN KEY, USERS, ID)"`
	NotificationId int `borm:"(NAME, notification_id) (FOREIGN KEY, NOTIFICATIONS, ID)"`
}

func main() {
	for _, arg := range os.Args {
		switch arg {
		case "--debug":
			borm.Settings().Environment().SetEnvironment(borm.DEBUGGING)
		case "--migrate":
			borm.Settings().Migrations().Enable().RecreateExisting().UndoOnError()
		}
	}

	// Registers default postgres database to migrate stuff through it
	commiter, err := borm.Connect(borm.RegisterDatabase("postgres", "db", borm.RegisterUser("postgres", "noeeekr")))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Create new database environments
	DEVELOPMENT_USER := commiter.RegisterUser("DEVELOPER", "developer")
	DEVELOPMENT_DATABASE := commiter.RegisterDatabase("DEVELOPMENT", DEVELOPMENT_USER)
	err = commiter.MigrateUsers(DEVELOPMENT_USER)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	development, err := commiter.MigrateDatabase(DEVELOPMENT_DATABASE)
	defer development.DB().Close()

	// Create new database relations
	USER_ROLES := development.RegisterEnum("user_role", string(STUDENT), string(TEACHER), string(ADMIN))
	TABLE_USERS := development.RegisterTable(Users{}).NeedRoles(USER_ROLES)
	TABLE_NOTIFICATIONS := development.RegisterTable(Notifications{}).NeedTables(TABLE_USERS)
	TABLE_USERS_NOTIFICATIONS := development.RegisterTable(UsersNotifications{}).Name("users_notifications").NeedTables(TABLE_USERS, TABLE_NOTIFICATIONS)
	// DEVELOPMENT_USER.GrantPrivileges(TABLE_USERS, borm.ALL)

	err = development.Relations()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	transaction, err := development.StartTx()
	if err != nil {
		fmt.Println(err)
		return
	}
	var issuerId int
	err = transaction.Do(TABLE_USERS.
		Insert("email", "password", "name", "role").
		Values("noeeekr@gmail.com", "noeeekr", "noeeekr", ADMIN).
		Returning("id").Scanner(scanInt(&issuerId)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	var targetId int
	err = transaction.Do(TABLE_USERS.
		Insert("email", "password", "name", "role").
		Values("cardozoandre0101@gmail.com", "andre", "andre", STUDENT).
		Returning("id").Scanner(scanInt(&targetId)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	var notificationId int
	err = transaction.Do(TABLE_NOTIFICATIONS.
		Insert("issuer_id", "title", "description").
		Values(issuerId, "test notification title", "test notification description").
		Returning("id").Scanner(scanInt(&notificationId)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = transaction.Do(TABLE_USERS_NOTIFICATIONS.
		Insert("user_id", "notification_id").
		Values(targetId, notificationId),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = transaction.Commit()
	if err != nil {
		fmt.Println(err)
		return
	}
}

func scanInt(i *int) borm.QueryRowsScanner {
	return func(rows *sql.Rows, throwErrorOnFound bool) *borm.Error {
		for rows.Next() {
			if err := rows.Scan(i); err != nil {
				return borm.NewError(err.Error()).Status(borm.ErrFailedOperation)
			}
		}
		err := rows.Close()
		if err != nil {
			return borm.NewError(err.Error()).Status(borm.ErrFailedOperation)
		}
		return nil
	}
}

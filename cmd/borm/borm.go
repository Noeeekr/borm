package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Noeeekr/borm"
	"github.com/Noeeekr/borm/common"
	"github.com/Noeeekr/borm/internal/registers"
)

type Users struct {
	Id int `borm:"(TYPE, SERIAL) (CONSTRAINTS, PRIMARY KEY)"`

	Name     string `borm:"(CONSTRAINTS, NOT NULL)"`
	Email    string `borm:"(CONSTRAINTS, NOT NULL)"`
	Password string `borm:"(CONSTRAINTS, NOT NULL)"`

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

	/*
		execution order:
			ENVIRONMENT
				- users
					- with password
					- with database
				- databases (using users)
					- owner user
			RELATIONS
				- roles (using databases and users)
				- tables (using databases and users)
					- privileged users
				- queries (using databases and users)

		database := borm.NewDatabase(*sql.DB)
		pipolo := database.User("piplo")
		pipole := database.User("piple")

		pizzas := database.Enum("pizzas", "chocolate", "cheese")

		INSTITUTIONS_TABLE := database.
		Table(Institutions{})

		USERS_TABLE := database.
			Table(Users{}).                           (Needed for queries)
			NeedTables(Institutions{})

		NOTIFICATIONS_TABLE := database.
			Table(Notifications{}).
			NeedTables(Users{}, Institutions{}).          (2 out of 3 already in cache)
			NeedRoles(piple, piplo, pizzas)

		pipolo.
			GrantPrivileges(USERS_TABLE).				 (If privilege list empty all privileges)
			ToColumns()        							 (If column list empty all columns)

		database.Migrate.Environment()
		database.Migrate.Relations()
	*/

	postgres, err := borm.On("postgres", "noeeekr", "db", "postgres")
	if err != nil {
		fmt.Println(err)
	}
	// Create new database environments
	DEVELOPMENT_USER := postgres.Register.User("DEVELOPER", "developer")
	DEVELOPMENT_DATABASE := postgres.NewDatabase("DEVELOPMENT", DEVELOPMENT_USER)
	CONFIGURATION := borm.NewConfiguration().RecreateExisting().UndoOnError()
	development, err := postgres.Environment(DEVELOPMENT_DATABASE, CONFIGURATION)
	if err != nil {
		fmt.Println(err.String())
		return
	}
	defer development.DB().Close()

	// Create new database relations
	LEVEL := development.Register.Enum("LEVEL", "JUNIOR", "PLENO", "SENIOR")
	TABLE_USERS := development.Register.Table(Users{}).NeedRoles(LEVEL)
	TABLE_NOTIFICATIONS := development.Register.Table(Notifications{}).NeedTables(TABLE_USERS)
	TABLE_USERS_NOTIFICATIONS := development.Register.Table(UsersNotifications{}).Name("users_notifications").NeedTables(TABLE_USERS, TABLE_NOTIFICATIONS)
	// DEVELOPMENT_USER.GrantPrivileges(TABLE_USERS, borm.ALL)

	err = development.Relations(CONFIGURATION)
	if err != nil {
		fmt.Println(err)
		return
	}

	transaction, err := development.Start()
	if err != nil {
		fmt.Println(err)
		return
	}
	var issuerId int
	err = transaction.Do(TABLE_USERS.
		Insert("email", "password", "name").
		Values("noeeekr@gmail.com", "noeeekr", "noeeekr").
		Returning("id").Scanner(scanInt(&issuerId)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	var targetId int
	err = transaction.Do(TABLE_USERS.
		Insert("email", "password", "name").
		Values("cardozoandre0101@gmail.com", "andre", "andre").
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

func scanInt(i *int) registers.QueryRowsScanner {
	return func(rows *sql.Rows, throwErrorOnFound bool) *common.Error {
		for rows.Next() {
			if err := rows.Scan(i); err != nil {
				return common.NewError(err.Error()).Status(common.ErrFailedOperation)
			}
		}
		err := rows.Close()
		if err != nil {
			return common.NewError(err.Error()).Status(common.ErrFailedOperation)
		}
		return nil
	}
}

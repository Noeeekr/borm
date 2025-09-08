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
	*Id

	Name     string   `borm:"(CONSTRAINTS, NOT NULL)"`
	Email    string   `borm:"(CONSTRAINTS, NOT NULL, UNIQUE)"`
	Password string   `borm:"(CONSTRAINTS, NOT NULL)"`
	Role     UserRole `borm:"(CONSTRAINTS, NOT NULL) (TYPE, user_role)"`
	// borm:"TYPE<serial> NAME<issuer_id> CONSTRAINTS<ignore, not null, unique, default 'abcd'> "
	// borm:"(TYPE, SERIAL) (CONSTRAINTS, PRIMARY KEY)"
	// borm:"type is serial, constraints are primary key and not null and default 'abcd', name is issuer_id"
	// borm:"type:serial, constraints:primary key, not null, default 'abcd', name:issuer_id"
	// borm:"type(serial) constraints(primary key, not null, default 'abcd') name(issuer_id)"

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
	USER_ROLES := development.RegisterEnum("user_role", STUDENT, TEACHER, ADMIN)
	TABLE_USERS := development.RegisterTable(Users{}).NeedRoles(USER_ROLES)
	TABLE_NOTIFICATIONS := development.RegisterTable(Notifications{}).NeedTables(TABLE_USERS)
	TABLE_USERS_NOTIFICATIONS := development.RegisterTable(UsersNotifications{}).Name("users_notifications").NeedTables(TABLE_USERS, TABLE_NOTIFICATIONS)
	// DEVELOPMENT_USER.GrantPrivileges(TABLE_USERS, borm.ALL)

	err = development.MigrateRelations()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	transaction, err := development.StartTx()
	if err != nil {
		fmt.Println(err)
		return
	}
	var firstUser int

	err = transaction.Do(TABLE_USERS.
		Insert("email", "password", "name", "role").
		Values("noeeekr@gmail.com", "noeeekr", "noeeekr", ADMIN).
		Returning("id").Scanner(scanInt(&firstUser)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	var secondUser int
	err = transaction.Do(TABLE_USERS.
		Insert("email", "password", "name", "role").
		Values(
			"email1@gmail.com", "123456", "andre", STUDENT,
			"email2@gmail.com", "123456", "peter", STUDENT,
			"email3@gmail.com", "123456", "gustav", STUDENT,
			"email4@gmail.com", "123456", "mason", STUDENT,
			"email5@gmail.com", "123456", "jorge", STUDENT,
			"email6@gmail.com", "123456", "alfredo", STUDENT,
		).
		Returning("id").Scanner(scanInt(&secondUser)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	var notificationId int
	err = transaction.Do(TABLE_NOTIFICATIONS.
		Insert("issuer_id", "description").
		Values(firstUser, "test notification description").
		Returning("id").Scanner(scanInt(&notificationId)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = transaction.Do(TABLE_USERS_NOTIFICATIONS.
		Insert("user_id", "notification_id").
		Values(firstUser, notificationId),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = transaction.Do(TABLE_NOTIFICATIONS.
		Insert("issuer_id", "title", "description").
		Values(firstUser, "test notification title 2", "test notification description 2").
		Returning("id").Scanner(scanInt(&notificationId)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = transaction.Do(TABLE_USERS_NOTIFICATIONS.
		Insert("user_id", "notification_id").
		Values(firstUser, notificationId),
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

	var notifications []*Notifications
	err = development.Do(TABLE_USERS.
		Select("n.id", "n.title", "n.description").As("u").
		InnerJoin(TABLE_USERS_NOTIFICATIONS, "un").On("u.id", "un.user_id").
		InnerJoin(TABLE_NOTIFICATIONS, "n").On("n.id", "un.notification_id").
		Where("u.email").Equals("noeeekr@gmail.com").
		OrderAscending("n.id").
		Scanner(scanNotifications(&notifications)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	var userAmountFound int
	err = development.Do(TABLE_USERS.
		Select("id", "email", "name").
		Where("email").Equals("noeeekr@gmail.com").
		OrderAscending("id").
		Scanner(RowAmount(&userAmountFound)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	var whereInExpectedReturn = []any{"peter", "andre", "jorge"}
	var whereInFoundAmount int
	err = development.Do(TABLE_USERS.
		Select("id", "email", "name").
		Where("name").In(whereInExpectedReturn...).
		OrderAscending("id").
		Scanner(RowAmount(&whereInFoundAmount)),
	)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("[Query Using Where (A) IN (A, B, C, ...)]")
	fmt.Println("\t[Expected Amount]:", len(whereInExpectedReturn))
	fmt.Println("\t[Found Amount]:", whereInFoundAmount)

	fmt.Println("[Issuer ID Returned From Insert]: ", firstUser)
	fmt.Println("[Notification Rows found]: ", len(notifications))
	for _, notification := range notifications {
		fmt.Println("\t[Notification ID]: ", notification.Id)
	}
}

func scanInt(i *int) borm.ReturnScanner {
	return func(rows *sql.Rows) (bool, error) {
		defer rows.Close()

		if rows.Next() {
			if err := rows.Scan(i); err != nil {
				return false, err
			}
		} else {
			return false, borm.ErrorDescription(borm.ErrNotFound, "No rows found")
		}

		return true, nil
	}
}

func scanNotifications(n *[]*Notifications) borm.ReturnScanner {
	return func(rows *sql.Rows) (bool, error) {
		defer rows.Close()

		var found bool = false
		for rows.Next() {
			notification := &Notifications{}
			rows.Scan(&notification.Id, &notification.Title, &notification.Description)
			*n = append(*n, notification)
			found = true
		}
		if rows.Err() != nil {
			return false, borm.ErrorDescription(borm.ErrUnexpected, rows.Err().Error())
		}
		return found, nil
	}
}

func RowAmount(i *int) borm.ReturnScanner {
	return func(rows *sql.Rows) (bool, error) {
		defer rows.Close()
		for rows.Next() {
			*i++
		}
		if *i == 0 {
			return false, nil
		}
		return true, nil
	}
}

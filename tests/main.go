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

	Name     string    `borm:"(CONSTRAINTS, NOT NULL)"`
	Email    string    `borm:"(CONSTRAINTS, NOT NULL, UNIQUE)"`
	Password string    `borm:"(CONSTRAINTS, NOT NULL)"`
	Role     *UserRole `borm:"(TYPE, user_role)"`

	SpecificWordA string `borm:"(NAME, specific_a)"`
	SpecificWordB string `borm:"(NAME, specific_b)"`
	SpecificWordC string `borm:"(NAME, specific_c)"`
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
type FilterClassOptions struct {
	A string
	B string
	C string
	D string
	E UserRole
	F *time.Time
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
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Create new database environments
	DEVELOPMENT_USER := commiter.RegisterUser("DEVELOPER", "developer")
	DEVELOPMENT_DATABASE := commiter.RegisterDatabase("DEVELOPMENT", DEVELOPMENT_USER)
	err = commiter.MigrateUsers(DEVELOPMENT_USER)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	development, err := commiter.MigrateDatabase(DEVELOPMENT_DATABASE)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer development.DB().Close()

	// Create new database relations
	USER_ROLES := development.RegisterEnum("user_role", STUDENT, TEACHER, ADMIN)
	TABLE_USERS := development.RegisterTable(Users{}).NeedRoles(USER_ROLES)
	TABLE_NOTIFICATIONS := development.RegisterTable(Notifications{}).NeedTables(TABLE_USERS)
	TABLE_USERS_NOTIFICATIONS := development.RegisterTable(UsersNotifications{}).Name("users_notifications").NeedTables(TABLE_USERS, TABLE_NOTIFICATIONS)
	// DEVELOPMENT_USER.GrantPrivileges(TABLE_USERS, borm.ALL)

	err = development.MigrateRelations()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	transaction, err := development.StartTx()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	var firstUser int

	err = transaction.Do(TABLE_USERS.
		Insert("email", "password", "name", "role").
		Values("noeeekr@gmail.com", "noeeekr", "noeeekr", ADMIN).
		Returning("id").Scanner(scanInt(&firstUser)),
	)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var nullRoleUser int
	err = transaction.Do(TABLE_USERS.
		Insert("email", "password", "name", "role").
		Values("noroleuser@gmail.com", "noroleuser", "noroleuser", nil).
		Returning("id").Scanner(scanInt(&nullRoleUser)),
	)
	if err != nil {
		fmt.Println(err.Error())
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
		fmt.Println(TABLE_USERS.
			Insert("email", "password", "name", "role").
			Values(
				"email1@gmail.com", "123456", "andre", STUDENT,
				"email2@gmail.com", "123456", "peter", STUDENT,
				"email3@gmail.com", "123456", "gustav", STUDENT,
				"email4@gmail.com", "123456", "mason", STUDENT,
				"email5@gmail.com", "123456", "jorge", STUDENT,
				"email6@gmail.com", "123456", "alfredo", STUDENT,
			).
			Returning("id").Scanner(scanInt(&secondUser)).CurrentValues...,
		)
		fmt.Println(err.Error())
		return
	}
	var notificationId int
	err = transaction.Do(TABLE_NOTIFICATIONS.
		Insert("issuer_id", "title", "description").
		Values(firstUser, "test notification title 1", "test notification description 1").
		Returning("id").Scanner(scanInt(&notificationId)),
	)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = transaction.Do(TABLE_USERS_NOTIFICATIONS.
		Insert("user_id", "notification_id").
		Values(firstUser, notificationId),
	)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = transaction.Do(TABLE_NOTIFICATIONS.
		Insert("issuer_id", "title", "description").
		Values(firstUser, "test notification title 2", "test notification description 2").
		Returning("id").Scanner(scanInt(&notificationId)),
	)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	composeTestUserName := "peter parker"
	composeTestUserEmail := "peter email"
	composeTestUserRole := STUDENT
	composeTestExpectedPassword := "Password Found"
	err = transaction.Do(TABLE_USERS.
		Insert("name", "email", "role", "specific_a", "specific_b", "specific_c", "password").
		Values(composeTestUserName, composeTestUserEmail, composeTestUserRole, "a", "b", "c", composeTestExpectedPassword),
	)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = transaction.Do(TABLE_USERS_NOTIFICATIONS.
		Insert("user_id", "notification_id").
		Values(firstUser, notificationId),
	)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	err = transaction.Commit()
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var notifications []*Notifications
	query := TABLE_USERS.
		Select("n.id", "n.title", "n.description").As("u").
		Scanner(scanNotifications(&notifications))

	query.
		InnerJoin(TABLE_USERS_NOTIFICATIONS, "un").On("u.id", "un.user_id").
		InnerJoin(TABLE_NOTIFICATIONS, "n").On("n.id", "un.notification_id").
		Where(
			query.And(
				query.Field("u.email").IsLike("%noe%", false),
				query.Compose(
					query.And(
						query.Field("u.email").IsEqual("noeeekr@gmail.com"),
						query.Field("n.title").IsAny("test notification title 2", "test notification title 1"),
						query.Field("n.title").IsLike("%notification%", false)),
				),
				query.Field("n.title").IsLike("%TEST%", false),
			),
		).
		OrderAscending("n.id")

	err = development.Do(query)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	var userAmountFound int
	query = TABLE_USERS.
		Select("id", "email", "name").
		Scanner(RowAmount(&userAmountFound))
	query.
		Where(query.Field("email").IsEqual("noeeekr@gmail.com")).
		OrderAscending("id")
	err = development.Do(query)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	composedTestPassword := ""

	query = TABLE_USERS.
		Select("password").
		Scanner(scanString(&composedTestPassword))
	query.Where(
		query.And(
			query.Compose(
				query.Field("name").IsEqual(composeTestUserName),
			),
			query.Compose(
				query.And(
					query.Compose(
						query.And(
							query.Field("email").IsEqual(composeTestUserEmail),
							query.Field("email").IsEqual(composeTestUserEmail),
						),
					),
					query.Compose(
						query.And(
							query.Field("name").IsEqual(composeTestUserName),
							query.Field("name").IsEqual(composeTestUserName),
						),
					),
				),
			),
		),
	)
	err = development.Do(query)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	var whereInExpectedReturn = []any{"peter", "andre", "jorge"}
	var whereInFoundAmount int
	query = TABLE_USERS.
		Select("id", "email", "name").
		Scanner(RowAmount(&whereInFoundAmount))
	query.
		Where(query.Field("name").IsAny(whereInExpectedReturn...)).
		OrderAscending("id")

	err = development.Do(query)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var nullUserName string
	query = TABLE_USERS.
		Select("name").
		Scanner(scanString(&nullUserName))
	query.Where(query.Field("role").IsEqual(nil))

	err = development.Do(query)
	if err != nil || nullUserName == "" {
		fmt.Println(err.Error())
	}

	name := ""
	query = TABLE_USERS.
		Select("c.name").As("c").
		InnerJoin(TABLE_USERS, "uc").On("c.name", "uc.name").
		InnerJoin(TABLE_USERS, "u").On("uc.name", "u.name").
		Scanner(scanString(&name))

	filters := []FilterClassOptions{
		{
			A: "andre",
			B: "andre",
			C: "andre",
			D: "andre",
			E: STUDENT,
			F: &time.Time{},
		},
	}

	conditions := make([]*borm.ConditionalQuery, len(filters))
	for i, filter := range filters {
		innerConditionals := []*borm.ConditionalQuery{}
		if filter.A == "" {
			innerConditionals = append(innerConditionals, query.Field("c.name").IsLike("%", false))
		} else {
			innerConditionals = append(innerConditionals, query.Field("c.name").IsLike("%"+filter.A+"%", false))
		}
		if filter.B != "" {
			innerConditionals = append(innerConditionals, query.Compose(
				query.And(
					query.Field("u.name").IsEqual(filter.B),
					query.Field("u.role").IsEqual(STUDENT),
				),
			))
		}
		if filter.C != "" {
			innerConditionals = append(
				innerConditionals,
				query.Compose(
					query.And(
						query.Field("u.name").IsEqual(filter.C),
						query.Field("u.role").IsEqual(STUDENT),
					),
				),
			)
		}
		if filter.D != "" {
			innerConditionals = append(innerConditionals, query.Field("c.name").IsEqual(filter.D))
		}
		if filter.E != "" {
			innerConditionals = append(innerConditionals, query.Field("c.role").IsEqual(filter.E))
		}
		if filter.F != nil {
			// query.And("c.created_at").After(filter.CreationYear) => Which is time.Time()
		}
		conditions[i] = query.Compose(query.And(innerConditionals...))
	}

	query.Where(
		query.And(
			query.Field("c.name").IsEqual("andre"),
			query.Compose(
				query.Or(conditions...),
			),
		),
	)

	if err := development.Do(query); err != nil {
		fmt.Println(err.Error())
		return
	}

	fmt.Println("")
	if composeTestExpectedPassword != composedTestPassword {
		fmt.Println("Failed to find correct password")
		fmt.Println("Expected: ", composeTestExpectedPassword)
		fmt.Println("Got: ", composedTestPassword)
		return
	} else {
		fmt.Println("[Composed test password found]: ", composedTestPassword)
	}
	if name != "andre" {
		fmt.Println("[Failed on compose test]")
		fmt.Println("\t[Expected name]:", "andre")
		fmt.Println("\t[Retrieved name]:", name)
		return
	} else {
		fmt.Println("[Sucessfull composed test]: Names match")
	}
	fmt.Println("[Query Using Where (A) IN (A, B, C, ...)]")
	fmt.Println("\t[Expected Amount]:", len(whereInExpectedReturn))
	fmt.Println("\t[Found Amount]:", whereInFoundAmount)
	fmt.Println("[Query SELECT name WHERE role IS NULL]")
	fmt.Println("\t[user with null role name]:", nullUserName)
	fmt.Println("[Issuer ID Returned From Insert]: ", firstUser)
	fmt.Println("[Notification Rows found]: ", len(notifications))
	for _, notification := range notifications {
		fmt.Println("\t[Notification ID]: ", notification.Id)
	}
}

func scanString(str *string) borm.ReturnScanner {
	return func(rows *sql.Rows) (bool, error) {
		defer rows.Close()

		if rows.Next() {
			if err := rows.Scan(str); err != nil {
				return false, err
			}
		} else {
			return false, borm.ErrorDescription(borm.ErrNotFound, "No rows found")
		}

		return true, nil
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

// func printBlockCondition(query *borm.Query) {
// 	blocks := []string{}
// 	for _, blockinfo := range query.Blocks {
// 		blocks = append(blocks, blockinfo.Block)
// 	}
// 	fmt.Println("-------")
// 	fmt.Println("[" + strings.Join(blocks, "]\n[") + "]")
// 	fmt.Println("-------")
// }

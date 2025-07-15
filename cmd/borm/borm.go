package main

import (
	"fmt"
	"time"

	"database/sql"

	"github.com/Noeeekr/borm"
)

type Users struct {
	Id int `type:"SERIAL" constraints:"PRIMARY KEY"`

	Name     string `constraints:"NOT NULL"`
	Email    string `constraints:"NOT NULL"`
	Password string `constraints:"NOT NULL"`

	DeletedAt time.Time `as:"deleted_at"`
	UpdatedAt time.Time `as:"updated_at"`
	CreatedAt time.Time `as:"created_at"`
}
type Notifications struct {
	id         int `borm:"(NAME, pip) (FOREIGN KEY, USERS, ID)"`
	issuerId   int `borm:"(NAME, issuer_id)"`
	targetUser int `borm:"(NAME, target_user) (FOREIGN KEY, USERS, ID)"`
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

	db, e := sql.Open("postgres", "postgres://postgres:noeeekr@db/postgres?sslmode=disable")
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	e = db.Ping()
	if e != nil {
		fmt.Println(e.Error())
		return
	}
	defer db.Close()
	postgres, err := borm.On("postgres", "noeeekr", "db", "postgres")
	if err != nil {
		fmt.Println(err)
	}
	// Create new database environments
	DEVELOPMENT_USER := postgres.User("DEVELOPER", "developer")
	DEVELOPMENT_DATABASE := postgres.NewDatabase("DEVELOPMENT", DEVELOPMENT_USER)
	CONFIGURATION := borm.NewConfiguration().RecreateExisting().UndoOnError()
	development, err := postgres.Environment(DEVELOPMENT_DATABASE, CONFIGURATION)
	if err != nil {
		fmt.Println(err.String())
		return
	}
	defer development.DB().Close()

	// Create new database relations
	LEVEL := development.Enum("LEVEL", "JUNIOR", "PLENO", "SENIOR")
	TABLE_USERS := development.Table(Users{}).NeedRoles(LEVEL)
	TABLE_NOTIFICATIONS := development.Table(Notifications{}).NeedTables(TABLE_USERS)
	development.Relations()
	fmt.Println(
		"database name and owner:", DEVELOPMENT_DATABASE.Name, DEVELOPMENT_DATABASE.Owner,
	)
	fmt.Println(
		TABLE_USERS.
			Select("id", "email", "name").
			Where("email", "noeeekr@gmail.com").Where("id", 10).
			Query,
	)
	fmt.Println(
		TABLE_USERS.
			Insert(TABLE_USERS, "id", "email", "name").
			Values(100, "100", "100").
			Query,
	)
	fmt.Println(
		TABLE_NOTIFICATIONS.
			Delete().
			Where("pip", 10).
			Query,
	)
	fmt.Println(
		TABLE_NOTIFICATIONS.
			Update().
			Where("pip", 10).
			Set("pip", 999).
			Query,
	)
	fmt.Println(
		TABLE_USERS.
			Update().
			Where("id", 10011).
			Set("id", 100).
			Set("name", "peter").
			Query,
	)
}

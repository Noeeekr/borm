package main

import (
	"fmt"
	"time"

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
	database := borm.NewDatabase(nil)

	TABLE_USERS := database.Table(Users{})
	TABLE_NOTIFICATIONS := database.Table(Notifications{}).NeedTables(TABLE_USERS)

	NOEEEKR := database.User("NOEEEKR THE DESTROYER")
	NOEEEKR.
		GrantPrivileges(TABLE_USERS, borm.INSERT, borm.DELETE, borm.UPDATE).
		ToColumns("id", "name")

	database.Migrate.Environment()
	database.Migrate.Relations()

	fmt.Println(
		borm.
			Select(TABLE_USERS, "id", "email", "name").
			Where("email", "noeeekr@gmail.com").Where("id", 10).
			Query,
	)
	fmt.Println(
		borm.
			Insert(TABLE_USERS, "id", "email", "name").
			Values(100, "100", "100").
			Query,
	)
	fmt.Println(
		borm.
			Delete(TABLE_NOTIFICATIONS).
			Where("pip", 10).
			Query,
	)
	fmt.Println(
		borm.
			Update(TABLE_NOTIFICATIONS).
			Where("pip", 10).
			Set("pip", 999).
			Query,
	)
	fmt.Println(
		borm.
			Update(TABLE_USERS).
			Where("id", 10011).
			Set("id", 100).
			Set("name", "peter").
			Query,
	)
}

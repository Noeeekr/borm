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
	response := borm.PrepareToCreate(
		Users{},
		Notifications{},
	)
	if response != nil {
		fmt.Println(response.String())
	}
	for _, query := range borm.GetCreateQueries() {
		fmt.Println(query)
	}

	fmt.Println(
		borm.
			Select(Users{}, "id", "email", "name").
			Where("email", "noeeekr@gmail.com").Where("id", 10).
			Query,
	)
	fmt.Println(
		borm.
			Insert(Users{}, "id", "email", "name").
			Values(100, "100", "100").
			Query,
	)
	fmt.Println(
		borm.
			Delete(Users{}).
			Where("id", 10).
			Query,
	)
	fmt.Println(
		borm.
			Update(Users{}).
			Where("id", 10).
			Set("id", time.Now()).
			Query,
	)
}

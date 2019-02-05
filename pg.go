package main

import (
	"database/sql"
	"log"
	"sync"
)

// Ensure pg implements client
var _ client = &pg{}

type pg struct {
	*sql.DB

	init       sync.Once
	dataSource string

	linkService pgLinkService
	userService pgUserService
}

func newPg(dataSource string) *pg {
	c := &pg{dataSource: dataSource}

	return c
}

// it would be wise to use this method with sync.Once
func (c *pg) initPg() {
	log.Println("initPg")
	db, err := sql.Open("postgres", c.dataSource)
	if err != nil {
		panic("cannot connect to database")
	}
	if err = db.Ping(); err != nil {
		panic("cannot ping to database")
	}

	c.DB = db
	c.linkService.Client = c
	c.userService.Client = c
}

func (c *pg) LinkService() linkService {
	c.init.Do(func() {
		c.initPg()
	})
	return &c.linkService
}

func (c *pg) UserService() userService {
	c.init.Do(func() {
		c.initPg()
	})
	return &c.userService
}

var _ userService = &pgUserService{}

type pgUserService struct {
	Client *pg
}

var _ linkService = &pgLinkService{}

type pgLinkService struct {
	Client *pg
}

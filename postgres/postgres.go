package postgres

import (
	"database/sql"
	"log"
	"sync"

	// Use postgresql.
	_ "github.com/lib/pq"
)

type Client struct {
	*sql.DB

	init       sync.Once
	dataSource string
}

// NewClient returns a new Postgres client.
func NewClient(dataSource string) *Client {
	c := Client{dataSource: dataSource}

	return &c
}

func (c *Client) initClient() {
	log.Println("initPostgres")
	db, err := sql.Open("postgres", c.dataSource)
	if err != nil {
		panic(err)
	}
	if err = db.Ping(); err != nil {
		panic("cannot ping to database")
	}

	c.DB = db
}

func (c *Client) cursor() *Client {
	c.init.Do(func() {
		c.initClient()
	})

	return c
}

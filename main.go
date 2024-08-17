package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	g2h "git.cmcode.dev/cmcode/ghost-to-hugo/lib"
	_ "github.com/go-sql-driver/mysql"
)

var flagConfig string

func parseFlags() {
	flag.StringVar(&flagConfig, "c", "config.json", "json file to use for loading configuration")
	flag.Parse()
}

func main() {
	parseFlags()

	c, err := g2h.LoadConfig(flagConfig)
	if err != nil {
		log.Fatalf("failed to load config: %v", err.Error())
	}

	db, err := sql.Open("mysql", c.MySQLConnectionString)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err.Error())
	}

	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	// force read-only since this operation does not require write privileges
	_, err = db.Exec("SET SESSION TRANSACTION READ ONLY")
	if err != nil {
		log.Fatalf("failed to set read-only session: %v", err.Error())
	}

	rows, err := db.Query(fmt.Sprintf("SELECT %v FROM posts", g2h.QUERY_POSTS_FIELDS))
	if err != nil {
		log.Fatalf("failed to query posts from db: %v", err.Error())
	}

	defer rows.Close()

	for rows.Next() {
		post, err := c.GetGhostPost(rows)
		if err != nil {
			log.Fatalf("failed to get ghost post from row: %v", err.Error())
		}

		if !c.IsValid(post) {
			log.Printf("skipping post %v", post.Title)
			continue
		}

		n, f, err := c.RenderOne(post)
		if err != nil {
			log.Fatalf("failed to render post in main loop: %v", err.Error())
		}

		log.Printf("wrote %v to %v (%v)", n, f, post.Title)

	}
}

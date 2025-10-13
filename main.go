package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/JGrinovich/BlogAggregator/internal/config"
	"github.com/JGrinovich/BlogAggregator/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	c := &commands{handlers: make(map[string]func(*state, command) error)}
	cfg, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DbUrl)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	dbQueries := database.New(db)

	s := &state{db: dbQueries, cfg: cfg}

	c.register("login", handlerLogin)
	c.register("register", handlerRegister)
	c.register("reset", handlerReset)
	c.register("users", handlerList)
	c.register("agg", handlerAgg)
	c.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	c.register("feeds", handlerFeedsList)
	c.register("follow", middlewareLoggedIn(handlerFollow))
	c.register("following", middlewareLoggedIn(handlerFollowing))
	c.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	c.register("browse", handlerBrowse)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "not enough arguments")
		os.Exit(1)
	}
	name := os.Args[1]
	args := []string{}
	if len(os.Args) > 2 {
		args = os.Args[2:]
	}
	cmd := command{name: name, args: args}

	if err := c.run(s, cmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

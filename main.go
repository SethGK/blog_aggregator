package main

import (
	"database/sql"
	"fmt"
	"gator/internal/cli"
	"gator/internal/config"
	"log"
	"os"

	_ "github.com/lib/pq"

	"gator/internal/database"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: not enough arguments provided")
		os.Exit(1)
	}

	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("error reading config: %v", err)
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	dbQueries := database.New(db)

	state := &cli.State{
		Config: &cfg,
		DB:     dbQueries,
	}

	commands := cli.NewCommands()
	commands.Register("register", cli.HandlerRegister)
	commands.Register("login", cli.HandlerLogin)
	commands.Register("reset", cli.HandlerReset)
	commands.Register("users", cli.HandlerUsers)
	commands.Register("agg", cli.HandlerAgg)
	commands.Register("addfeed", cli.MiddlewareLoggedIn(cli.HandlerAddFeedLogged))
	commands.Register("feeds", cli.HandlerFeeds)
	commands.Register("follow", cli.MiddlewareLoggedIn(cli.HandlerFollowLogged))
	commands.Register("following", cli.MiddlewareLoggedIn(cli.HandlerFollowingLogged))
	commands.Register("unfollow", cli.MiddlewareLoggedIn(cli.HandlerUnfollowLogged))
	commands.Register("browse", cli.MiddlewareLoggedIn(cli.HandlerBrowsePostsLogged))

	cmd := cli.Command{
		Name: os.Args[1],
		Args: os.Args[2:],
	}

	if err := commands.Run(state, cmd); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

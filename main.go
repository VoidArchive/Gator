package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"

	"github.com/voidarchive/Gator/internal/cli"
	"github.com/voidarchive/Gator/internal/config"
	"github.com/voidarchive/Gator/internal/database"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal("Error reading config:", err)
	}

	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	dbQueries := database.New(db)

	programState := &cli.State{
		Cfg: &cfg,
		DB:  dbQueries,
	}

	cmds := cli.NewCommands()
	cmds.Register("login", cli.HandlerLogin)
	cmds.Register("register", cli.HandlerRegister)
	cmds.Register("reset", cli.HandlerReset)
	cmds.Register("users", cli.HandlerUsers)
	cmds.Register("agg", cli.HandlerAgg)

	args := os.Args
	if len(args) < 2 {
		fmt.Println("Error: not enough arguments provided")
		os.Exit(1)
	}

	cmd := cli.Command{
		Name: args[1],
		Args: args[2:],
	}

	err = cmds.Run(programState, cmd)
	if err != nil {
		fmt.Printf("Error :%v\n", err)
		os.Exit(1)
	}
}

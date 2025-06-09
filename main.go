package main

import (
	"fmt"
	"log"
	"os"

	"github.com/voidarchive/Gator/internal/cli"
	"github.com/voidarchive/Gator/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		log.Fatal("Error reading config:", err)
	}
	programState := &cli.State{
		Cfg: &cfg,
	}

	cmds := cli.NewCommands()
	cmds.Register("login", cli.HandlerLogin)

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

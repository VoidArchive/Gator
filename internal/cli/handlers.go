package cli

import "fmt"

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) == 0 {
		return fmt.Errorf("login requires a username")
	}
	username := cmd.Args[0]
	if err := s.Cfg.SetUser(username); err != nil {
		return err
	}
	fmt.Printf("User has been set to: %s\n", username)
	return nil
}

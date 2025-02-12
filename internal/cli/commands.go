package cli

import (
	"errors"
	"fmt"

	"gator/internal/config"
)

type State struct {
	Config *config.Config
}

type Command struct {
	Name string
	Args []string
}

type HandlerFunc func(s *State, cmd Command) error

type Commands struct {
	handlers map[string]HandlerFunc
}

func NewCommands() *Commands {
	return &Commands{
		handlers: make(map[string]HandlerFunc),
	}
}

func (c *Commands) Register(name string, f HandlerFunc) {
	c.handlers[name] = f
}

func (c *Commands) Run(s *State, cmd Command) error {
	handler, exists := c.handlers[cmd.Name]
	if !exists {
		return fmt.Errorf("unknown comand: %s", cmd.Name)
	}
	return handler(s, cmd)
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) < 1 {
		return errors.New("username is required")
	}
	username := cmd.Args[0]

	err := s.Config.SetUser(username)
	if err != nil {
		return err
	}

	fmt.Printf("User set to %s\n", username)
	return nil
}

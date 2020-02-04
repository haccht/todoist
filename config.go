package todoist

import (
	"fmt"

	"github.com/tucnak/store"
)

const configFile = "todoist.json"

type Config struct {
	Token  string `json:"token"`
	Filter string `json:"filter,omitempty"`
	Closed uint   `json:"closed,omitempty"`
}

func NewConfig() (*Config, error) {
	var config Config

	store.Init("todoist")
	if err := store.Load(configFile, &config); err != nil {
		return nil, err
	}

	if config.Token == "" {
		fmt.Printf("Input API Token: ")
		fmt.Scan(&config.Token)
		config.Save()
	}

	return &config, nil
}

func (c *Config) Save() {
	store.Save(configFile, c)
}

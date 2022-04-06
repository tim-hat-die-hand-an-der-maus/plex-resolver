package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
)

type Config struct {
	Servers []ConfigPlexServer `toml:"server"`
}

type ConfigPlexServer struct {
	Name  string `toml:"name"`
	Url   string `toml:"url"`
	Token string `toml:"token"`
}

func ReadConfig(filename string) (*Config, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	_, err = toml.Decode(string(content), &config)

	var result Config
	for _, server := range config.Servers {
		if server.Url == "" || server.Name == "" || server.Token == "" || server.Token == "illegal base64 data at input byte 0" {
			fmt.Printf("ignoring server: %v\n", server)
		} else {
			result.Servers = append(result.Servers, server)
		}
	}

	return &result, err
}

func ConfigServerToResponseServer(servers []ConfigPlexServer, responses []MovieResponse) []ResponseServer {
	result := make([]ResponseServer, len(servers)-1)
	for idx, server := range servers {
		result = append(result, ResponseServer{
			Name:  server.Name,
			Url:   server.Url,
			Error: responses[idx].Error,
		})
	}

	return result
}

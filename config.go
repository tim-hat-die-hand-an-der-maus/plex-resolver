package main

import (
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
		if server.Url == "" || server.Name == "" || server.Token == "" {
			result.Servers = append(result.Servers, server)
		}
	}

	return &result, err
}

func ConfigServerToResponseServer(servers []ConfigPlexServer) []ResponseServer {
	result := make([]ResponseServer, len(servers))
	for _, server := range servers {
		result = append(result, ResponseServer{
			Name: server.Name,
			Url:  server.Url,
		})
	}

	return result
}

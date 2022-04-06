package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"time"
)

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		log.Fatalf("env variable %s not found", name)
	}

	return value
}

func plexServerResponse(server ConfigPlexServer) MovieResponse {
	plex := New(server.Url, server.Token)
	movies, err := plex.Movies()
	if err != nil {
		err = fmt.Errorf("failed to retrieve movies: %v", err)
	}

	return MovieResponse{
		Name:   server.Name,
		Movies: movies,
		Error:  &err,
	}
}

func isAllError(serverResponse []MovieResponse) bool {
	if len(serverResponse) == 0 {
		return true
	}

	for _, movie := range serverResponse {
		if *movie.Error != nil {
			return true
		}
	}

	return false
}

func main() {
	configFilename := os.Getenv("CONFIG_FILENAME")
	if configFilename == "" {
		configFilename = "config.toml"
	}

	config, err := ReadConfig(configFilename)
	if err != nil {
		fmt.Printf("failed to read config:\n%v\n", err)
		return
	}

	r := gin.Default()
	r.GET("/movies", func(c *gin.Context) {
		response := make([]MovieResponse, len(config.Servers)-1)

		for _, server := range config.Servers {
			response = append(response, plexServerResponse(server))
		}

		if isAllError(response) {
			c.Header("Content-Type", "application/problem+json")
			c.Header("Content-Language", "en")

			c.JSON(500, gin.H{
				"data":    make([]int, 0),
				"type":    "https://github.com/tim-hat-die-hand-an-der-maus/plex-resolver",
				"title":   "Failed to retrieve movies",
				"detail":  fmt.Sprintf("Failed to retrieve movies from all %d plex servers", len(config.Servers)),
				"servers": ConfigServerToResponseServer(config.Servers, response),
			})
		} else {
			c.JSON(200, gin.H{
				"data": response,
			})
		}
	})

	log.Fatal(r.Run("0.0.0.0:8080"))
}

func New(baseUrl, token string) Plex {
	client := http.Client{
		Timeout: time.Second * 30,
	}

	return Plex{
		baseUrl: baseUrl,
		token:   token,
		client:  client,
	}
}

func videosToMovies(videos []Video) []Movie {
	var movies []Movie

	for _, video := range videos {
		movies = append(movies, video.ToMovie())
	}

	return movies
}

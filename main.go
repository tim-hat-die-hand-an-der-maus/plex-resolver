package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
	"strconv"
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
	movies, err := plex.DirectoryContentByName("movie")
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

func movies(c *gin.Context, config *Config) (int, gin.H) {
	response := make([]MovieResponse, len(config.Servers)-1)

	for _, server := range config.Servers {
		response = append(response, plexServerResponse(server))
	}

	if isAllError(response) {
		c.Header("Content-Type", "application/problem+json")
		c.Header("Content-Language", "en")

		return 500, gin.H{
			"data":    []int{},
			"type":    "https://github.com/tim-hat-die-hand-an-der-maus/plex-resolver",
			"title":   "Failed to retrieve movies",
			"detail":  fmt.Sprintf("Failed to retrieve movies from all %d plex servers", len(config.Servers)),
			"servers": ConfigServerToResponseServer(config.Servers, response),
		}
	} else {
		return 200, gin.H{
			"data": response,
		}
	}
}

func moviesAddedSince(c *gin.Context, config *Config) (int, gin.H) {
	code, content := movies(c, config)
	if code != 200 {
		return code, content
	}
	data := content["data"].([]MovieResponse)

	sinceParam := c.Param("since")
	title, detail := "", ""
	if sinceParam == "" {
		title = "no `since` parameter is set in URL (`/since/<unix-timestamp>`)"
		detail = fmt.Sprintf("the current path `%s` does not contain a since (`/since/<unix-timestamp>`) parameter", c.Request.URL.Path)
	}
	since, err := strconv.Atoi(sinceParam)
	if err != nil {
		title = "the `since` parameter couldn't be parsed as an integer (`/since/<unix-timestamp>`)"
		detail = fmt.Sprintf("the `since` parameter (`%s` | `%s`) couldn't be parsed as an integer: %v", c.Request.URL.Path, sinceParam, err)
	}
	if len(title) > 0 || len(detail) > 0 {
		return 400, gin.H{
			"data":    make([]int, 0),
			"type":    "https://github.com/tim-hat-die-hand-an-der-maus/plex-resolver",
			"title":   title,
			"detail":  detail,
			"servers": ConfigServerToResponseServer(config.Servers, data),
		}
	}

	responses := make([]MovieResponse, 0)
	for _, server := range data {
		movies := make([]Movie, 0)
		for _, movie := range server.Movies {
			if movie.AddedAt >= uint64(since) {
				movies = append(movies, movie)
			}
		}

		if len(movies) > 0 {
			server.Movies = movies
			responses = append(responses, server)
		}
	}

	return 200, gin.H{
		"data": responses,
	}
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
		code, content := movies(c, config)
		c.JSON(code, content)
	})
	r.GET("/movies/:since", func(c *gin.Context) {
		code, content := moviesAddedSince(c, config)
		c.JSON(code, content)
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

func directoriesToMovies(videos []Directory) []Movie {
	var movies []Movie

	for _, video := range videos {
		movies = append(movies, video.ToMovie())
	}

	return movies
}

func videosToMovies(videos []Video) []Movie {
	var movies []Movie

	for _, video := range videos {
		movies = append(movies, video.ToMovie())
	}

	return movies
}

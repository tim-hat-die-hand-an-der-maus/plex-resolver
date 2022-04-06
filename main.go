package main

import (
	"encoding/xml"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
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
	config, err := ReadConfig("config.toml")
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

func (v Video) ToMovie() Movie {
	return Movie{
		Title: v.Title,
	}
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

func (p Plex) Get(url string, marshalInto interface{}) (*http.Response, error) {
	url = p.baseUrl + "/" + url + "?X-Plex-Token="
	// TODO: use url type for this
	url = url + p.token

	response, err := p.client.Get(url)
	if err != nil {
		//goland:noinspection GoUnusedCallResult
		fmt.Errorf("failed to retrieve url: %s", err)
		return response, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		//goland:noinspection GoUnusedCallResult
		err = fmt.Errorf("couldn't read from response body %v", err)
		return response, err
	}

	if response.StatusCode != 200 {
		err = fmt.Errorf("httpcode != 200:\n\t%s\n", body)
		return nil, err
	}

	err = xml.Unmarshal(body, &marshalInto)
	if err != nil {
		err = fmt.Errorf("couldn't unmarshal body: %s", err)
		return nil, err
	}

	return response, nil
}

func (p Plex) Libraries() (*MediaContainer, error) {
	var container MediaContainer
	_, err := p.Get("library/sections", &container)
	if err != nil {
		err = fmt.Errorf("failed to get library/sections: %s", err)
		return nil, err
	}

	return &container, nil
}

func (p Plex) Library(id string) (*MediaContainerLibrary, error) {
	var container MediaContainerLibrary
	url := fmt.Sprintf("library/sections/%s/all", id)

	_, err := p.Get(url, &container)
	if err != nil {
		err = fmt.Errorf("failed to get %s\n", url)
		return nil, err
	}

	return &container, nil
}

func videosToMovies(videos []Video) []Movie {
	var movies []Movie

	for _, video := range videos {
		movies = append(movies, video.ToMovie())
	}

	return movies
}

func (p Plex) Movies() ([]Movie, error) {
	libraries, err := p.Libraries()
	if err != nil {
		log.Printf("Failed to retrieve libraries: %v\n", err)
		return make([]Movie, 0), err
	}

	var videos []Movie

	for _, directory := range libraries.Directories {
		if directory.Type != "movie" {
			continue
		}

		library, err := p.Library(directory.Location.Id)
		if err != nil {
			return videos, fmt.Errorf("failed to retrieve library [%s] %s", directory.Location.Id, directory.Title)
		}

		videos = append(videos, videosToMovies(library.Videos)...)
	}

	return videos, nil
}

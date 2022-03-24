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

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func requiredEnv(name string) string {
	value := os.Getenv(name)
	if len(value) == 0 {
		log.Fatalf("env variable %s not found", name)
	}

	return value
}

type Movie struct {
	Title string `json:"title"`
}

type QueueRequest struct {
	Action  string `json:"action"`
	Request string `json:"request"`
	Queue   string `json:"queue"`
}

func main() {
	token := requiredEnv("PLEX_TOKEN")
	plexUrl := requiredEnv("PLEX_URL")
	r := gin.Default()
	r.GET("/movies", func(c *gin.Context) {
		plex := New(plexUrl, token)
		movies, err := plex.Movies()
		if err != nil {
			c.JSON(500, gin.H{
				"movies": make([]int, 0),
				"error":  fmt.Sprintf("%v", err),
			})
		} else {
			c.JSON(200, gin.H{
				"movies": movies,
			})
		}
	})

	log.Fatal(r.Run("0.0.0.0:8080"))
}

type Plex struct {
	baseUrl string
	token   string
	client  http.Client
}

type Location struct {
	XMLName xml.Name `xml:"Location"`
	Id      string   `xml:"id,attr"`
}

type Directory struct {
	XMLName  xml.Name `xml:"Directory"`
	Title    string   `xml:"title,attr"`
	Type     string   `xml:"type,attr"`
	Location Location `xml:"Location"`
}

type MediaContainer struct {
	XMLName     xml.Name    `xml:"MediaContainer"`
	Size        string      `xml:"size,attr"`
	ViewGroup   string      `xml:"viewGroup,attr"`
	Directories []Directory `xml:"Directory"`
}

type Video struct {
	XMLName xml.Name `xml:"Video"`
	Title   string   `xml:"title,attr"`
}

func (v Video) ToMovie() Movie {
	return Movie{
		Title: v.Title,
	}
}

type MediaContainerLibrary struct {
	XMLName xml.Name `xml:"MediaContainer"`
	Size    string   `xml:"size,attr"`
	Videos  []Video  `xml:"Video"`
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
	failOnError(err, "failed to retrieve libraries")

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

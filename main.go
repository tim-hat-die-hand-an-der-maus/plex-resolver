package main

import (
	"encoding/xml"
	"fmt"
	"github.com/streadway/amqp"
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

func main() {
	token := requiredEnv("PLEX_TOKEN")
	plexUrl := requiredEnv("PLEX_URL")
	amqpUser := requiredEnv("AMQP_USER")
	amqpPassword := requiredEnv("AMQP_PASSWORD")
	amqpHost := requiredEnv("AMQP_HOST")
	amqpPort := requiredEnv("AMQP_PORT")

	plex := New(plexUrl, token)
	amqpUrl := fmt.Sprintf("amqp://%s:%s@%s:%s", amqpUser, amqpPassword, amqpHost, amqpPort)
	conn, err := amqp.Dial(amqpUrl)
	//goland:noinspection GoUnhandledErrorResult
	defer conn.Close()
	if err != nil {
		log.Fatalf("rabbitmq dial failed (%s): %s\n", amqpUrl, err)
	}

	ch, err := conn.Channel()
	failOnError(err, "failed to create channel")
	//goland:noinspection GoUnhandledErrorResult
	defer ch.Close()

	queue, err := ch.QueueDeclare(requiredEnv("QUEUE_NAME"), false, false, false, false, nil)
	failOnError(err, "failed to declare queue")
	movies, err := plex.Movies()
	failOnError(err, "failed to retrieve movies")

	for _, video := range movies {
		err = ch.Publish(
			"",
			queue.Name,
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(video.Title),
			})

		failOnError(err, "failed to publish message")
	}
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
	fmt.Printf("calling %s\n", url)
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

	fmt.Printf("unmarshal body:\n\t%s\n", string(body))
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

func (p Plex) Movies() ([]Video, error) {
	libraries, err := p.Libraries()
	failOnError(err, "failed to retrieve libraries")

	var videos []Video

	for _, directory := range libraries.Directories {
		if directory.Type != "movie" {
			continue
		}

		library, err := p.Library(directory.Location.Id)
		if err != nil {
			return videos, fmt.Errorf("failed to retrieve library [%s] %s", directory.Location.Id, directory.Title)
		}
		videos = append(videos, library.Videos...)
	}

	return videos, nil
}

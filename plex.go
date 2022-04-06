package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type Plex struct {
	baseUrl string
	token   string
	client  http.Client
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

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

func (p Plex) Directories() ([]Directory, error) {
	libraries, err := p.Libraries()
	if err != nil {
		log.Printf("Failed to retrieve libraries: %v\n", err)
		return []Directory{}, err
	}

	var directories []Directory

	for _, directory := range libraries.Directories {
		directories = append(directories, directory)
	}

	return directories, nil
}

// FIXME: take a conversion function (e.g. videosToMovies) - also use a generic type for the return
//        + access to library (e.g. library.Videos)
func (p Plex) DirectoryContent(directory Directory) ([]Movie, error) {
	library, err := p.Library(directory.Location.Id)
	if err != nil {
		return []Movie{}, fmt.Errorf("failed to retrieve library [%s] %s", directory.Location.Id, directory.Title)
	}

	if directory.Type == "movie" {
		return videosToMovies(library.Videos), nil
	} else {
		return directoriesToMovies(library.Directories), nil
	}
}

// FIXME: Use a generic return type + conversion function (see comment for DirectoryContent)
func (p Plex) DirectoryContentByName(name string) ([]Movie, error) {
	directories, err := p.Directories()
	if err != nil {
		log.Printf("Failed to retrieve directories: %v\n", err)
		return []Movie{}, err
	}

	var videos []Movie
	for _, directory := range directories {
		if directory.Type != name {
			continue
		}

		movies, err := p.DirectoryContent(directory)
		if err != nil {
			return nil, err
		}
		videos = append(videos, movies...)
		break
	}

	return videos, nil
}

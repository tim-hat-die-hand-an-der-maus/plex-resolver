package main

import (
	"encoding/xml"
	"strconv"
)

type Movie struct {
	Title   string  `json:"title"`
	Year    *uint16 `json:"year"`
	AddedAt uint64  `json:"added-at"`
}

type ResponseServer struct {
	Name  string `json:"name"`
	Url   string `json:"url"`
	Error *error `json:"error"`
}

type MovieResponse struct {
	Name   string  `json:"name"`
	Movies []Movie `json:"movies"`
	Error  *error  `json:"error"`
}

type QueueRequest struct {
	Action  string `json:"action"`
	Request string `json:"request"`
	Queue   string `json:"queue"`
}

type Location struct {
	XMLName xml.Name `xml:"Location"`
	Id      string   `xml:"id,attr"`
}

type Directory struct {
	XMLName  xml.Name `xml:"Directory"`
	Title    string   `xml:"title,attr"`
	Year     string   `xml:"year,attr"`
	Type     string   `xml:"type,attr"`
	Location Location `xml:"Location"`
	AddedAt  int64    `xml:"addedAt,attr"`
}

func (d Directory) ToMovie() Movie {
	var year *uint16

	_year, err := strconv.ParseInt(d.Year, 10, 16)
	if err != nil {
		year = nil
	} else {
		y := uint16(_year)
		year = &y
	}

	return Movie{
		Title:   d.Title,
		Year:    year,
		AddedAt: uint64(d.AddedAt),
	}
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
	Year    string   `xml:"year,attr"`
	AddedAt int64    `xml:"addedAt,attr"`
}

func (v Video) ToMovie() Movie {
	var year *uint16

	_year, err := strconv.ParseInt(v.Year, 10, 16)
	if err != nil {
		year = nil
	} else {
		y := uint16(_year)
		year = &y
	}

	return Movie{
		Title:   v.Title,
		Year:    year,
		AddedAt: uint64(v.AddedAt),
	}
}

type MediaContainerLibrary struct {
	XMLName     xml.Name    `xml:"MediaContainer"`
	Size        string      `xml:"size,attr"`
	Videos      []Video     `xml:"Video"`
	Directories []Directory `xml:"Directory"` // for TV-Shows
}

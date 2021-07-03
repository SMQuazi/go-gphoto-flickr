package main

import (
	"encoding/json"
	"io/ioutil"
)

type Albums struct {
	Albums []Album `json:"albums"`
}

type Album struct {
	PhotoCount  string   `json:"photo_count"`
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ViewCount   string   `json:"view_count"`
	Created     string   `json:"created"`
	LastUpdated string   `json:"last_updated"`
	CoverPhoto  string   `json:"cover_photo"`
	Photos      []string `json:"photos"`
}

func CheckError(e error) {
	if e != nil {
		panic(e)
	}
}

func ReadFlickrAlbums(file string) Albums {
	data, err := ioutil.ReadFile(file)
	CheckError(err)

	var albums Albums
	CheckError(json.Unmarshal(data, &albums))

	return albums
}

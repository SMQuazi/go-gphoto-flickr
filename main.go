package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	gphotos "github.com/gphotosuploader/google-photos-api-client-go/v2"
	"golang.org/x/oauth2"
)

type Config struct {
	PhotoPath string `json:"PhotoPath"`
}

func ReadConfig() Config {
	data, err := ioutil.ReadFile("./config.json")
	CheckError(err)

	var config Config
	CheckError(json.Unmarshal(data, &config))

	return config
}

func FindFiles(pattern string, root string) ([]string, error) {
	var matchedFiles []string

	filepath.WalkDir(root, func(path string, dir fs.DirEntry, err error) error {
		matched, matchErr := filepath.Match("*"+pattern+"*", filepath.Base(path))
		if matchErr != nil {
			return matchErr
		}
		if matched {
			matchedFiles = append(matchedFiles, path)
		}
		return nil
	})

	return matchedFiles, nil
}

func main() {
	config := ReadConfig()
	albums := ReadFlickrAlbums("./albums.json")

	// Create http.Client for gphoto
	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     "1057053981746-o7gqj7lpf80h4h3ogjvj5pco9v8ea6pi.apps.googleusercontent.com",
		ClientSecret: "nQGtZ2Y_RVOufDzfAq5maWcB",
	}
	var code string
	if _, err := fmt.Scan(&code); err != nil {
		log.Fatal(err)
	}
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		log.Fatal(err)
	}
	tc := conf.Client(ctx, tok)

	//Use authenticated http to start
	client, err := gphotos.NewClient(tc)

	for _, album := range albums.Albums {
		fmt.Println(album.Title)

		safeAlbumTitle := strings.Replace(album.Title, "/", "_", -1)
		safeAlbumTitle = strings.Replace(safeAlbumTitle, "\\", "_", -1)
		safeAlbumTitle = strings.Replace(safeAlbumTitle, ":", "_", -1)
		safeAlbumTitle = strings.Replace(safeAlbumTitle, "?", "_", -1)
		safeAlbumTitle = strings.Replace(safeAlbumTitle, "<", "_", -1)
		safeAlbumTitle = strings.Replace(safeAlbumTitle, ">", "_", -1)
		safeAlbumTitle = strings.Replace(safeAlbumTitle, "|", "_", -1)
		safeAlbumTitle = strings.Trim(safeAlbumTitle, " ")

		client.Albums.Create(ctx, safeAlbumTitle)

		destDir := filepath.Join(config.PhotoPath, safeAlbumTitle)
		os.Mkdir(destDir, os.ModeDir)

		for _, photo := range album.Photos {
			fmt.Printf("searching for %s...", photo)
			files, _ := FindFiles(photo, config.PhotoPath)

			if len(files) == 1 {
				fmt.Printf("found %d. %s\n", len(files), files)
				destFile := filepath.Join(destDir, filepath.Base(files[0]))
				fmt.Println("  Moving to", destFile)
				moveErr := os.Rename(files[0], destFile)

				if moveErr != nil {
					log.Fatal(moveErr)
				}
			}
		}
	}
}

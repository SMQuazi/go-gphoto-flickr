package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	gphotos "github.com/gphotosuploader/google-photos-api-client-go/v2"
	"golang.org/x/oauth2"
)

type Config struct {
	PhotoPath     string `json:"PhotoPath"`
	GPhotoAPIPath string `json:"GPhotoAPIPath"`
}

type GPhotoApi struct {
	Installed Installed `json:"installed"`
}
type Installed struct {
	ClientID                string   `json:"client_id"`
	ProjectID               string   `json:"project_id"`
	AuthURI                 string   `json:"auth_uri"`
	TokenURI                string   `json:"token_uri"`
	AuthProviderX509CertURL string   `json:"auth_provider_x509_cert_url"`
	RedirectUris            []string `json:"redirect_uris"`
}

func SetEnvironment() (Config, GPhotoApi) {
	ConfigByte, err := ioutil.ReadFile("./config.json")
	CheckError(err)

	var config Config
	CheckError(json.Unmarshal(ConfigByte, &config))

	gPhotoByte, err := ioutil.ReadFile(config.GPhotoAPIPath)
	CheckError(err)
	s := string(gPhotoByte)
	fmt.Println(s)

	var gPhotoApi GPhotoApi
	jsonErr := json.Unmarshal(gPhotoByte, &gPhotoApi)
	if jsonErr != nil {
		panic(jsonErr)
	}
	s = string(gPhotoByte)
	fmt.Println(s)

	return config, gPhotoApi
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
	config, gphotoinfo := SetEnvironment()
	albums := ReadFlickrAlbums("./albums.json")

	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     gphotoinfo.Installed.ClientID,
		ClientSecret: "EL4Z3wQNZNaBz25SS5eOD8Qj",
		Scopes: []string{
			"https://www.googleapis.com/auth/photoslibrary.appendonly",
			"https://www.googleapis.com/auth/photoslibrary.readonly.appcreateddata",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  gphotoinfo.Installed.AuthURI,
			TokenURL: gphotoinfo.Installed.TokenURI,
		},
		RedirectURL: gphotoinfo.Installed.RedirectUris[0],
	}

	var code string
	fmt.Printf("Access URI: %s\n", conf.AuthCodeURL("state", oauth2.AccessTypeOffline))
	fmt.Print("input: ")
	_, err := fmt.Scan(&code)
	CheckError(err)

	fmt.Println("Creating token")
	tok, err := conf.Exchange(ctx, code)
	CheckError(err)

	fmt.Println("Creating client")
	tc := conf.Client(ctx, tok)

	// Use authenticated http to start
	client, err := gphotos.NewClient(tc)
	CheckError(err)

	// Sift through albums in JSON
	for _, album := range albums.Albums {
		fmt.Println(album.Title)

		// Set the safe name for folder and gphoto albums
		var safeAlbumTitle = album.Title
		for _, char := range []string{"/", "\\", ":", "?", "<", ">", "|"} {
			safeAlbumTitle = strings.Replace(safeAlbumTitle, char, "_", -1)
		}
		safeAlbumTitle = strings.Trim(safeAlbumTitle, " ")

		fmt.Printf("Creating album %s\n", safeAlbumTitle)
		gPhotoAlbum, err := client.Albums.Create(ctx, safeAlbumTitle)
		CheckError(err)

		// Create the folder if needed
		destDir := filepath.Join(config.PhotoPath, safeAlbumTitle)
		os.Mkdir(destDir, os.ModeDir)

		for _, photo := range album.Photos {
			// Search for files
			fmt.Printf("searching for %s...", photo)
			files, _ := FindFiles(photo, config.PhotoPath)

			if len(files) == 1 {
				fmt.Printf("found %d. %s\n", len(files), files)
				destFile := filepath.Join(destDir, filepath.Base(files[0]))

				// Move files if found
				fmt.Printf("  Moving to %s\n", destFile)
				CheckError(os.Rename(files[0], destFile))

				// Then upload
				fmt.Println("  Uploading $s", destFile)
				mediaItem, err := client.UploadFileToAlbum(ctx, gPhotoAlbum.ID, destFile)
				CheckError(err)
				fmt.Printf("  done. (%s)", mediaItem.ProductURL)

			}
		}
	}
}

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
	"time"

	"github.com/pkg/browser"

	gphotos "github.com/gphotosuploader/google-photos-api-client-go/v2"
	"github.com/gphotosuploader/google-photos-api-client-go/v2/albums"
	"golang.org/x/oauth2"
)

type FilePaths struct {
	PhotoDirPath       string `json:"PhotoDirPath"`
	GPhotoAuthJsonPath string `json:"GPhotoAuthJsonPath"`
	FlickrJsonPath     string `json:"FlickrJsonPath"`
}

// Check the error and panic, if any
func CheckError(e error) {
	if e != nil {
		panic(e)
	}
}

// Reads `./config.json` to grab all necessary paths
func GetAppConfig() FilePaths {
	ConfigByte, err := ioutil.ReadFile("./config.json")
	CheckError(err)

	var config FilePaths
	CheckError(json.Unmarshal(ConfigByte, &config))

	return config
}

// Find files in the `rootDir` (and sub directories) that match the `searchPattern`
func FindFiles(searchPattern string, rootDir string) ([]string, error) {
	var matchedFiles []string

	filepath.WalkDir(rootDir, func(path string, dir fs.DirEntry, err error) error {
		matched, matchErr := filepath.Match("*"+searchPattern+"*", filepath.Base(path))
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

// Sorts pictures in photoDir from flickr archive download into
// albums based on the albums.json included in the
// download
func SortLocalFlickrAlbums(photoDir string, FlickrJsonPath string) {

	// Go through albums in the Flickr JSON
	albums := ReadFlickrAlbums(FlickrJsonPath)

	for _, album := range albums.Albums {
		fmt.Println(album.Title)

		// Set the safe name for folder and gphoto albums
		var safeAlbumTitle = album.Title
		for _, char := range []string{"/", "\\", ":", "?", "<", ">", "|"} {
			safeAlbumTitle = strings.Replace(safeAlbumTitle, char, "_", -1)
		}
		safeAlbumTitle = strings.Trim(safeAlbumTitle, " ")

		// Create the folder if needed
		destDir := filepath.Join(photoDir, safeAlbumTitle)
		os.Mkdir(destDir, os.ModeDir)

		// Read through photos in the album
		for _, photo := range album.Photos {
			fmt.Printf("searching for %s...", photo)
			files, _ := FindFiles(photo, photoDir)

			// If single file is matched
			if len(files) == 1 {
				fmt.Printf("found %d. %s\n", len(files), files)
				destFile := filepath.Join(destDir, filepath.Base(files[0]))

				// Move the file
				fmt.Printf("  Moving to %s...", destFile)
				CheckError(os.Rename(files[0], destFile))
				fmt.Print("  done.\n", destFile)
			}
		}
	}
}

type ftgAlbums []albums.Album

// Checks whether an album with albumName exists
func (f ftgAlbums) ContainsAlbumNamed(albumName string) bool {
	for _, a := range f {
		if a.Title == albumName {
			return true
		}
	}

	return false
}

// Creates an album if it doesn't exist, or finds it if it does
func CreateAlbum(client gphotos.Client, string name) {

}

// Creates a folder & image structure on Google Photos
// mirroring one from a photoPath using gPhotoInfo to authenticate
func FromPCtoGPhotos(photoPath string, gPhotoInfo GPhotoAuth) error {
	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     gPhotoInfo.Installed.ClientID,
		ClientSecret: "EL4Z3wQNZNaBz25SS5eOD8Qj",
		Scopes: []string{
			"https://www.googleapis.com/auth/photoslibrary.appendonly",
			"https://www.googleapis.com/auth/photoslibrary.readonly.appcreateddata",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  gPhotoInfo.Installed.AuthURI,
			TokenURL: gPhotoInfo.Installed.TokenURI,
		},
		RedirectURL: gPhotoInfo.Installed.RedirectUris[0],
	}

	var code string
	browser.OpenURL(conf.AuthCodeURL("state", oauth2.AccessTypeOffline))
	fmt.Print("Paste code from Google now: ")
	_, err := fmt.Scanln(&code)
	if err != nil {
		return err
	}

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		return err
	}
	tc := conf.Client(ctx, tok)
	client, err := gphotos.NewClient(tc)
	if err != nil {
		return err
	}

	// Get list of albums to check against to avoid creating duplicates
	existingAlbums, err := client.Albums.List(ctx)
	ftgExistingAlbums := ftgAlbums(existingAlbums)
	if err != nil {
		return err
	}

	defaultAlbumName := "Flickr Unsorted " + time.Now().Format("yyyyMMddHHmm")
	filepath.WalkDir(photoPath, func(path string, entry fs.DirEntry, err error) error {

		// Get parent folder of each files
		relPath, err := filepath.Rel(photoPath, path)
		fmt.Println(relPath)
		localParentName, _ := filepath.Split(relPath)
		if localParentName == "" {
			localParentName = defaultAlbumName
		}

		var gPhotoAlbum *albums.Album
		if ftgExistingAlbums.ContainsAlbumNamed(localParentName) {
			// If it exists in gPhotos, use it
			gPhotoAlbum, err = client.Albums.GetByTitle(ctx, localParentName)

		} else {
			// if it doesn't, create one to use
			gPhotoAlbum, err = client.Albums.Create(ctx, localParentName)
			if err != nil {
				return err
			}
			ftgExistingAlbums = append(ftgExistingAlbums, *gPhotoAlbum)
		}

		// Upload file to folder
		uploadItem, err := client.UploadFileToAlbum(ctx, gPhotoAlbum.ID, path)
		return err

		return nil
	})

	return nil
}

func main() {
	config := GetAppConfig()

	SortLocalFlickrAlbums(config.PhotoDirPath, config.FlickrJsonPath)

	gPhotoAuth := GetGPhotoAuthFromFile(config.GPhotoAuthJsonPath)
	FromPCtoGPhotos(config.PhotoDirPath, gPhotoAuth)
}

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type GPhotoAuth struct {
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

// Reads the oauth2 file created/downloaded from the google cloud console
func GetGPhotoAuthFromFile(filePath string) GPhotoAuth {
	gPhotoByte, err := ioutil.ReadFile(filePath)
	CheckError(err)
	s := string(gPhotoByte)
	fmt.Println(s)

	var gPhotoApi GPhotoAuth
	jsonErr := json.Unmarshal(gPhotoByte, &gPhotoApi)
	if jsonErr != nil {
		panic(jsonErr)
	}

	return gPhotoApi
}

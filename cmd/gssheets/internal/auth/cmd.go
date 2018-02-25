package auth

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	flagSet    = flag.NewFlagSet("auth", flag.ExitOnError)
	credential = flagSet.String("credential", "client-credential.json", "Google OAuth Client Credential")
)

var (
	Name        = "auth"
	Description = "authorize Google Account"
	Usage       = flagSet.PrintDefaults
)

const (
	googleScope = "https://www.googleapis.com/auth/spreadsheets"
)

// Run command
func Run(args []string) error {
	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if len(*credential) == 0 {
		return fmt.Errorf("credential is required")
	}

	config, err := ConfigFromJSON(*credential)
	if err != nil {
		return err
	}

	tok, err := getTokenFromWeb(config)
	if err != nil {
		return err
	}

	cacheFile, err := tokenCacheFile()
	if err != nil {
		return err
	}

	if err := saveToken(cacheFile, tok); err != nil {
		return err
	}

	fmt.Println("~/.credentials/ has been created")
	return nil
}

// ConfigFromJSON load OAuth client credential
func ConfigFromJSON(file string) (*oauth2.Config, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Unable to read client secret file: %v", err)
	}

	return google.ConfigFromJSON(b, googleScope)
}

// TokenFromCache load OAuth token from cache
func TokenFromCache() (t *oauth2.Token, err error) {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		return nil, fmt.Errorf("Unable to get path to cached credential file. %v", err)
	}

	f, err := os.Open(cacheFile)
	if err != nil {
		return nil, err
	}

	t = &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer func() {
		if e := f.Close(); e != nil && err == nil {
			err = e
		}
	}()

	return t, err
}

func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	if err := os.MkdirAll(tokenCacheDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(tokenCacheDir, "sheets.googleapis.com-nlp-dictionaries.json"), nil
}

func saveToken(file string, token *oauth2.Token) (err error) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("Unable to cache oauth token: %v", err)
	}
	defer func() {
		if e := f.Close(); e != nil && err == nil {
			err = e
		}
	}()

	return json.NewEncoder(f).Encode(token)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	fmt.Printf("authorization code: ")

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return nil, fmt.Errorf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve token from web %v", err)
	}
	return tok, nil
}

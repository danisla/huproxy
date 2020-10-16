package main

/*
Acquire id_token for a user for a given audience
TO use, first download a client_secret.json for installed app flow (desktop):
   https://cloud.google.com/iap/docs/authentication-howto#authenticating_from_a_desktop_app

   specify the audience you would like
   run login flow on browser.  You refresh_token will be saved into credential_file so that you are not repeatedly running the login flows (be careful with this)

   go run oauth2oidc.go --audience=1071284184436-vu96hfaugnm9falak0pl00ur9cuvldl2.apps.googleusercontent.com --credential_file=creds.json --client_secrets_file=client_secret.json
*/

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jws"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

var (
	flCredentialFile    = flag.String("credential_file", "creds.json", "Credential File")
	flClientSecretsFile = flag.String("client_secrets_file", "client_secrets.json", "(required) tcp host:port to connect")
	flAudience          = flag.String("audience", "", "(required) Audience for the token")
)

const (
	userInfoEmailScope = "https://www.googleapis.com/auth/userinfo.email"
)

func main() {

	flag.Parse()
	if *flClientSecretsFile == "" {
		log.Fatalf("specify either --client_secrets_file must be set")
	}

	if *flAudience == "" {
		log.Fatalf("--audience must be set")
	}

	b, err := ioutil.ReadFile(*flClientSecretsFile)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	conf, err := google.ConfigFromJSON(b, userInfoEmailScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	var refreshToken string
	_, err = os.Stat(*flCredentialFile)
	if *flCredentialFile == "" || os.IsNotExist(err) {
		lurl := conf.AuthCodeURL("code")
		fmt.Printf("\nVisit the URL for the auth dialog and enter the authorization code  \n\n%s\n", lurl)
		fmt.Printf("\nEnter code:  ")
		input := bufio.NewScanner(os.Stdin)
		input.Scan()

		tok, err := conf.Exchange(oauth2.NoContext, input.Text())
		if err != nil {
			log.Fatalf("Cloud not exchange TOken %v", err)
		}
		refreshToken = tok.RefreshToken
	} else {
		f, err := os.Open(*flCredentialFile)
		if err != nil {
			log.Fatalf("Could not open credential File %v", err)
		}
		defer f.Close()
		tok := &tokenResponse{}
		err = json.NewDecoder(f).Decode(tok)
		if err != nil {
			log.Fatalf("Could not parse credential File %v", err)
		}
		refreshToken = tok.RefreshToken

		var parser *jwt.Parser
		parser = new(jwt.Parser)
		tt, _, err := parser.ParseUnverified(tok.IDToken, &jwt.StandardClaims{})
		if err != nil {
			log.Fatalf("Could not parse saved id_tokne File %v", err)
		}

		c, ok := tt.Claims.(*jwt.StandardClaims)
		err = tt.Claims.Valid()
		if ok && c.Audience == *flAudience && err == nil {
			fmt.Printf("%s\n", tt.Raw)
			return
		}

	}
	data := url.Values{
		"client_id":     {conf.ClientID},
		"client_secret": {conf.ClientSecret},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
		"audience":      {*flAudience},
	}

	resp, err := http.PostForm(conf.Endpoint.TokenURL, data)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Fatal(string(b))
	}

	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		log.Fatalf("oauth2: cannot fetch token: %v", err)
	}

	tokenRes := &tokenResponse{}

	if err := json.Unmarshal(body, tokenRes); err != nil {
		log.Fatalf("oauth2: cannot fetch token: %v", err)
	}
	token := &oauth2.Token{
		AccessToken: tokenRes.AccessToken,
		TokenType:   tokenRes.TokenType,
	}
	raw := make(map[string]interface{})
	json.Unmarshal(body, &raw) // no error checks for optional fields
	token = token.WithExtra(raw)

	if secs := tokenRes.ExpiresIn; secs > 0 {
		token.Expiry = time.Now().Add(time.Duration(secs) * time.Second)
	}
	if v := tokenRes.IDToken; v != "" {
		// decode returned id token to get expiry
		claimSet, err := jws.Decode(v)
		if err != nil {
			log.Fatalf("oauth2: error decoding JWT token: %v", err)
		}
		token.Expiry = time.Unix(claimSet.Exp, 0)
	}
	tokenRes.RefreshToken = refreshToken
	f, err := os.OpenFile(*flCredentialFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to save credential file: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(tokenRes)

	fmt.Printf("%s\n", tokenRes.IDToken)
}

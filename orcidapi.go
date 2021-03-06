package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

// ORCIDAPIURL is the string containing the scheme, host, and API version path for the
// ORCID API we want to use.
const ORCIDAPIURL string = "https://pub.orcid.org/v2.1/"

// AccessToken is returned to us from the ORCID API. Used for authentication.
type AccessToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// ORCIDSearchResponse stores data from the ORCID API when we do a /search call.
type ORCIDSearchResponse struct {
	Result []struct {
		OrcidIdentifier struct {
			URI  string `json:"uri"`
			Path string `json:"path"`
			Host string `json:"host"`
		} `json:"orcid-identifier"`
	} `json:"result"`
	NumFound int `json:"num-found"`
}

// ORCIDExternalIdentifierResult stores data from the ORCID API when we do a /[id]/external-identifiers call.
type ORCIDExternalIdentifierResult struct {
	LastModifiedDate struct {
		Value int64 `json:"value"`
	} `json:"last-modified-date"`
	ExternalIdentifier []struct {
		CreatedDate struct {
			Value int64 `json:"value"`
		} `json:"created-date"`
		LastModifiedDate struct {
			Value int64 `json:"value"`
		} `json:"last-modified-date"`
		Source struct {
			SourceOrcid    interface{} `json:"source-orcid"`
			SourceClientID struct {
				URI  string `json:"uri"`
				Path string `json:"path"`
				Host string `json:"host"`
			} `json:"source-client-id"`
			SourceName struct {
				Value string `json:"value"`
			} `json:"source-name"`
		} `json:"source"`
		ExternalIDType  string `json:"external-id-type"`
		ExternalIDValue string `json:"external-id-value"`
		ExternalIDURL   struct {
			Value string `json:"value"`
		} `json:"external-id-url"`
		ExternalIDRelationship string `json:"external-id-relationship"`
		Visibility             string `json:"visibility"`
		Path                   string `json:"path"`
		PutCode                int    `json:"put-code"`
		DisplayIndex           int    `json:"display-index"`
	} `json:"external-identifier"`
	Path string `json:"path"`
}

func getORCIDSearchToken() string {

	v := url.Values{}
	v.Set("client_id", *clientid)
	v.Set("client_secret", *clientsecret)
	v.Set("grant_type", "client_credentials")
	v.Set("scope", "/read-public")

	resp, err := http.PostForm("https://orcid.org/oauth/token", v)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("Unable to get access token from API.")
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}
		log.Fatalf("%v, %s", resp.StatusCode, bodyBytes)
	}

	var token AccessToken

	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		log.Fatalln(err)
	}

	return token.AccessToken
}

func findORCIDsFromAPIUsingScopus(orcids map[string]*IDInfo, scopusID, token string) {

	request, err := http.NewRequest("GET", ORCIDAPIURL+"search/?q=external-id-reference:"+scopusID, nil)
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Set("Accept", "application/vnd.orcid+json")
	request.Header.Set("Authorization", "Bearer "+token)

	log.Println(request)

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Println("Bad HTTP status from API.")
		log.Fatalf("%v, %s", resp.StatusCode, bodyBytes)
	}

	var response ORCIDSearchResponse

	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		log.Fatalln(err)
	}

	for _, result := range response.Result {
		if result.OrcidIdentifier.Host == "orcid.org" {
			if _, exists := orcids[result.OrcidIdentifier.Path]; !exists {
				orcids[result.OrcidIdentifier.Path] = &IDInfo{Processed: false, New: true}
			}
		}
	}

}

func findScopusIDsFromAPIUsingORCID(scopusIDs map[string]*IDInfo, orcid, token string) {

	request, err := http.NewRequest("GET", ORCIDAPIURL+orcid+"/external-identifiers", nil)
	if err != nil {
		log.Fatal(err)
	}
	request.Header.Set("Accept", "application/vnd.orcid+json")
	request.Header.Set("Authorization", "Bearer "+token)

	log.Println(request)

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Println("Bad HTTP status from API.")
		log.Fatalf("%v, %s", resp.StatusCode, bodyBytes)
	}

	var response ORCIDExternalIdentifierResult

	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		log.Fatalln(err)
	}

	for _, externalIdentifier := range response.ExternalIdentifier {
		if externalIdentifier.ExternalIDType == "Scopus Author ID" {
			if _, exists := scopusIDs[externalIdentifier.ExternalIDValue]; !exists {
				scopusIDs[externalIdentifier.ExternalIDValue] = &IDInfo{Processed: false, New: true}
			}
		}
	}

}

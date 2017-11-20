package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// Person is a cozy little struct for storing data about people from the Person-export.json files
type Person struct {
	FamilyName string `json:"family_name"`
	GivenName  string `json:"given_name"`
	ID         string `json:"__id__"`
	Identifier []struct {
		Scheme string `json:"scheme"`
		Value  string `json:"value"`
	} `json:"identifier"`
}

// IDInfo stores if an identifier is new, and if it has been processed yet.
type IDInfo struct {
	Processed bool
	New       bool
}

var clientid = flag.String("client_id", "", "Client ID for ORCID API")
var clientsecret = flag.String("client_secret", "", "Client Secret for ORCID API")

func findFilesToProcess() []string {
	if len(flag.Args()) == 0 {
		log.Println("No file names provided, trying to find files ending with Person-export.json in current working directory.")
		workingDir, err := os.Getwd()
		if err != nil {
			log.Fatalln("Error getting working directory. ", err)
		}
		matches, err := filepath.Glob(filepath.Join(workingDir, "*Person-export.json"))
		if err != nil {
			log.Fatalln("Error finding matching files. ", err)
		}
		return matches
	}

	return flag.Args()
}

func processFile(filename, token string) {

	file, err := os.Open(filename)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	fileScanner := bufio.NewScanner(file)

	counter := 0

	for fileScanner.Scan() {

		counter++
		log.Println(counter)

		var person Person

		err := json.Unmarshal(fileScanner.Bytes(), &person)
		if err != nil {
			log.Fatalln(err)
		}

		processPerson(person, token)
	}
}

func processPerson(person Person, token string) {

	orcids := map[string]*IDInfo{}
	scopusIDs := map[string]*IDInfo{}

	for _, identifier := range person.Identifier {
		if identifier.Scheme == "orcid" {
			orcids[identifier.Value] = &IDInfo{Processed: false, New: false}
		} else if identifier.Scheme == "scopus" {
			scopusIDs[identifier.Value] = &IDInfo{Processed: false, New: false}
		}
	}

	for {
		done := true
		for orcid, info := range orcids {
			if !info.Processed {
				findScopusIDsFromAPIUsingORCID(scopusIDs, orcid, token)
				info.Processed = true
				done = false
			}
		}
		for scopusID, info := range scopusIDs {
			if !info.Processed {
				findORCIDsFromAPIUsingScopus(orcids, scopusID, token)
				info.Processed = true
				done = false
			}
		}
		if done {
			break
		}
	}

	printOutput(person, orcids, scopusIDs)
}

func printOutput(person Person, orcids, scopusIDs map[string]*IDInfo) {
	printedPerson := false
	printedOrcid := false
	printedScopus := false
	for orcid, info := range orcids {
		if info.New {
			if !printedPerson {
				fmt.Println(person.GivenName, person.FamilyName, person.ID)
				printedPerson = true
			}
			if !printedOrcid {
				fmt.Println("New ORCIDs:")
				printedOrcid = true
			}
			fmt.Println(orcid)
		}
	}
	for scopusID, info := range scopusIDs {
		if info.New {
			if !printedPerson {
				fmt.Println(person.GivenName, person.FamilyName, person.ID)
				printedPerson = true
			}
			if !printedScopus {
				fmt.Println("New Scoups IDs:")
				printedScopus = true
			}
			fmt.Println(scopusID)
		}
	}
	if printedPerson {
		fmt.Println("")
	}
}

func main() {
	flag.Parse()

	filesToProcess := findFilesToProcess()
	if len(filesToProcess) == 0 {
		log.Fatalln("Could not find any files to process.")
	}

	if *clientid == "" {
		log.Fatalln("You need to provide a client_id")
	}

	if *clientsecret == "" {
		log.Fatalln("You need to provide a client_secret")
	}

	token := getORCIDSearchToken()

	for _, fileName := range filesToProcess {
		processFile(fileName, token)
	}

}

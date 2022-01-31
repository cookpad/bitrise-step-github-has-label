package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func requireStringEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("$%v required", key)
	}
	return value
}

func requireIntEnv(key string) int {
	strValue := requireStringEnv(key)
	value, err := strconv.Atoi(strValue)
	if err != nil {
		log.Fatalf("$%v must be an integer", key)
	}
	return value
}

func exportEnv(key, value string) {
	cmdLog, err := exec.Command("bitrise", "envman", "add", "--key", key, "--value", value).CombinedOutput()
	if err != nil {
		log.Fatalf("error exporting environment variable with envman: %#v | output: %s", err, cmdLog)
	}
}

// Function taken from https://github.com/bitrise-steplib/steps-github-status
// ownerAndRepo returns the owner and the repository part of a git repository url. Possible url formats:
// - https://hostname/owner/repository.git
// - git@hostname:owner/repository.git
func ownerAndRepo(url string) (string, string) {
	url = strings.TrimPrefix(strings.TrimPrefix(url, "https://"), "git@")
	a := strings.FieldsFunc(url, func(r rune) bool { return r == '/' || r == ':' })
	return a[1], strings.TrimSuffix(a[2], ".git")
}

type Label struct {
	Id   int
	Name string
}

func main() {
	apiBaseUrl := requireStringEnv("api_base_url")
	repositoryUrl := requireStringEnv("repository_url")
	owner, repo := ownerAndRepo(repositoryUrl)
	accessToken := requireStringEnv("access_token")
	issueNumber := requireIntEnv("issue_number")
	labelWanted := requireStringEnv("label")

	if !strings.HasSuffix(apiBaseUrl, "/") {
		apiBaseUrl += "/"
	}
	url := fmt.Sprintf("%vrepos/%v/%v/issues/%v/labels", apiBaseUrl, owner, repo, issueNumber)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("error creating request: ", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("token %v", accessToken))
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("error sending request: ", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("unexpected status code %v: %v", resp.StatusCode, string(body))
	}

	var labels []Label
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&labels)
	if err != nil {
		log.Fatal("error decoding JSON: ", err)
	}

	hasLabel := "false"
	for _, label := range labels {
		if label.Name == labelWanted {
			hasLabel = "true"
			break
		}
	}

	exportEnv("HAS_LABEL", hasLabel)
}

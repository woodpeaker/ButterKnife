package main

import (
	"encoding/json"
	"flag"
	"log"
	"math"
	"net/http"
	//"strconv"
	"io/ioutil"
	"strings"
	"time"
)

var (
	apiKey string
	host   string
)

type Id struct {
	Id int `json:"id"`
}

type IdName struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type issuesResult struct {
	Issues []Issue `json:"issues"`
}

type Issue struct {
	Id           int            `json:"id"`
	Subject      string         `json:"subject"`
	Description  string         `json:"description"`
	ProjectId    int            `json:"project_id"`
	Project      *IdName        `json:"project"`
	Tracker      *IdName        `json:"tracker"`
	StatusId     int            `json:"status_id"`
	Status       *IdName        `json:"status"`
	Priority     *IdName        `json:"priority"`
	Author       *IdName        `json:"author"`
	AssignedTo   *IdName        `json:"assigned_to"`
	Notes        string         `json:"notes"`
	StatusDate   string         `json:"status_date"`
	CreatedOn    string         `json:"created_on"`
	UpdatedOn    string         `json:"updated_on"`
	CustomFields []*CustomField `json:"custom_fields"`
}

type TimeEntry struct {
	Id        int    `json:"id"`
	Project   IdName `json:"project"`
	Issue     Id     `json:"issue"`
	User      IdName `json:"user"`
	Activity  IdName `json:"activity"`
	Hours     float32
	SpentOn   string `json:"spent_on"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
}

type NewTimeEntry struct {
	IssueId    int     `json:"issue_id"`
	SpentOn    string  `json:"spent_on"`
	Hours      float64 `json:"hours"`
	ActivityId int     `json:"activity_id"`
}

type timeEntriesResult struct {
	TimeEntries []TimeEntry `json:"time_entries"`
}

type timeEntryRequest struct {
	TimeEntry NewTimeEntry `json:"time_entry"`
}

type CustomField struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func init() {
	flag.StringVar(&apiKey, "apikey", "", "Redmine API Key")
	flag.StringVar(&host, "host", "", "Redmine host")
}

func myIssues(host string, apiKey string) issuesResult {
	path := "/issues.json?assigned_to_id=me&limit=100&key=" + apiKey
	url := "https://" + host + path
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	decoder := json.NewDecoder(res.Body)
	var r issuesResult
	if res.StatusCode != 200 {
		log.Println(res.StatusCode)
		log.Fatal(res)
	}
	err = decoder.Decode(&r)
	if err != nil {
		log.Println("decoding error")
		log.Fatal(err)
	}
	//log.Println(len(r.Issues))
	return r
}

func myTimeEntries(host string, apiKey string) timeEntriesResult {
	path := "/time_entries.json?user_id=me&sort=spent_on:desc&limit=100&key=" + apiKey
	url := "https://" + host + path
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	decoder := json.NewDecoder(res.Body)
	var r timeEntriesResult
	if res.StatusCode != 200 {
		log.Println(res.StatusCode)
		log.Fatal(res)
	}
	err = decoder.Decode(&r)
	if err != nil {
		log.Println("decoding error")
		log.Fatal(err)
	}
	//log.Println(len(r.TimeEntries))
	return r
}

func makeTimeEntry(host string, apiKey string, issueId int, today string, timeToAdd float64, projectId int) {
	path := "/time_entries.json?key=" + apiKey
	//path := "/projects/" + strconv.Itoa(projectId) + "/time_entries/new?key=" + apiKey
	url := "https://" + host + path

	var ir timeEntryRequest
	ir.TimeEntry = NewTimeEntry{IssueId: issueId,
		SpentOn: today,
		Hours:   timeToAdd, ActivityId: 9}
	s, err := json.Marshal(ir)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(s)))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Redmine-API-Key", apiKey)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(res)
	body, err := ioutil.ReadAll(res.Body)
	log.Printf("Body: %s", body)

	defer res.Body.Close()
}

func main() {
	flag.Parse()
	today := time.Now().Format("2006-01-02")
	log.Printf("Today is: %v\n", today)
	entries := myTimeEntries(host, apiKey)

	var trackedTime float32 = 0

	for _, timeEntry := range entries.TimeEntries {
		//log.Printf("so: %v h: %v", timeEntry.SpentOn, timeEntry.Hours)
		if timeEntry.SpentOn == today {
			trackedTime += timeEntry.Hours
		}
	}

	log.Printf("Tracked today: %v\n", trackedTime)

	var workHours float32 = 8

	if trackedTime < workHours {
		var missingHours float32 = workHours - trackedTime
		issues := myIssues(host, apiKey)
		var issuesCount int = len(issues.Issues)
		var timePerIssue float32 = missingHours / float32(issuesCount)
		var roundedTimePerIssue float64 = math.Floor(float64(timePerIssue*10)) / 10
		var extraTime float64 = float64(workHours) - (roundedTimePerIssue * float64(issuesCount))

		log.Printf("Missing hours: %v Issues: %v Time per issue: %v(rounded: %v) Extra time: %v\n", missingHours, issuesCount, timePerIssue, roundedTimePerIssue, extraTime)
		log.Printf("To track: %v", extraTime+(roundedTimePerIssue*float64(issuesCount)))

		for num, issue := range issues.Issues {

			timeToAdd := roundedTimePerIssue
			// Add extraTime to the first ticket
			if num == 0 {
				timeToAdd += extraTime
			}

			log.Printf("Tracking %v in #%v", timeToAdd, issue.Id)
			//makeTimeEntry(host, apiKey, issue.Id, today, timeToAdd, issue.Project.Id)
			//return
		}
	}
}

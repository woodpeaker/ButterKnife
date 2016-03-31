package main

import (
	"encoding/json"
	"flag"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

var (
	apiKey string
	host   string
	dryRun bool
)

type id struct {
	ID int `json:"id"`
}

type idName struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type issuesResult struct {
	Issues []issue `json:"issues"`
}

type issue struct {
	ID           int            `json:"id"`
	Subject      string         `json:"subject"`
	Description  string         `json:"description"`
	ProjectID    int            `json:"project_id"`
	Project      *idName        `json:"project"`
	Tracker      *idName        `json:"tracker"`
	StatusID     int            `json:"status_id"`
	Status       *idName        `json:"status"`
	Priority     *idName        `json:"priority"`
	Author       *idName        `json:"author"`
	AssignedTo   *idName        `json:"assigned_to"`
	Notes        string         `json:"notes"`
	StatusDate   string         `json:"status_date"`
	CreatedOn    string         `json:"created_on"`
	UpdatedOn    string         `json:"updated_on"`
	CustomFields []*customField `json:"custom_fields"`
}

type timeEntry struct {
	ID        int    `json:"id"`
	Project   idName `json:"project"`
	Issue     id     `json:"issue"`
	User      idName `json:"user"`
	Activity  idName `json:"activity"`
	Hours     float64
	SpentOn   string `json:"spent_on"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
}

type newTimeEntry struct {
	IssueID    int     `json:"issue_id"`
	SpentOn    string  `json:"spent_on"`
	Hours      float64 `json:"hours"`
	ActivityID int     `json:"activity_id"`
}

type timeEntriesResult struct {
	TimeEntries []timeEntry `json:"time_entries"`
}

type timeEntryRequest struct {
	TimeEntry newTimeEntry `json:"time_entry"`
}

type customField struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func init() {
	flag.StringVar(&apiKey, "apikey", "", "Redmine `APIKey`")
	flag.StringVar(&host, "host", "", "Redmine `HOST`")
	flag.BoolVar(&dryRun, "dry", false, "Dry run")
}

func myIssues(host string, apiKey string) (issues issuesResult) {
	url := "https://" + host + "/issues.json?assigned_to_id=me&limit=100&key=" + apiKey
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Println(res.StatusCode)
		log.Fatal(res)
	}

	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&issues)
	if err != nil {
		log.Println("decoding error")
		log.Fatal(err)
	}
	return
}

func myTimeEntries(host string, apiKey string) (entries timeEntriesResult) {
	url := "https://" + host + "/time_entries.json?user_id=me&sort=spent_on:desc&limit=100&key=" + apiKey
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Println(res.StatusCode)
		log.Fatal(res)
	}

	decoder := json.NewDecoder(res.Body)
	err = decoder.Decode(&entries)
	if err != nil {
		log.Println("decoding error")
		log.Fatal(err)
	}
	return
}

func makeTimeEntry(host string, apiKey string, issueID int, today string, timeToAdd float64) {
	url := "https://" + host + "/time_entries.json?key=" + apiKey

	var ir timeEntryRequest
	ir.TimeEntry = newTimeEntry{
		IssueID:    issueID,
		SpentOn:    today,
		Hours:      timeToAdd,
		ActivityID: 9,
	}
	data, err := json.Marshal(ir)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", url, strings.NewReader(string(data)))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Redmine-API-Key", apiKey)

	client := &http.Client{}
	res, err := client.Do(req)
	defer res.Body.Close()

	if err != nil {
		log.Fatal(err)
	}
	//log.Println(res)
	//body, err := ioutil.ReadAll(res.Body)
	//log.Printf("Body: %s", body)
}

func main() {
	var trackedTime = 0.0
	var workHours = 8.0

	flag.Parse()
	today := time.Now().Format("2006-01-02")
	log.Printf("Today is: %v\n", today)

	entries := myTimeEntries(host, apiKey)
	for _, timeEntry := range entries.TimeEntries {
		if timeEntry.SpentOn == today {
			trackedTime += timeEntry.Hours
		}
	}
	log.Printf("Tracked today: %v\n", trackedTime)

	if trackedTime < workHours {
		issues := myIssues(host, apiKey)

		var issuesCount = len(issues.Issues)
		var missingHours = workHours - trackedTime
		var timePerIssue = missingHours / float64(issuesCount)
		var roundedTimePerIssue = math.Floor(timePerIssue*10) / 10
		var extraTime = missingHours - (roundedTimePerIssue * float64(issuesCount))

		// min = 0.25
		// x = (missinHours / min)
		// rest = x % issuesCount
		// timePerIssue = x / issuesCount * min

		log.Printf("Missing hours: %v Issues: %v Time per issue: %v(rounded: %v) Extra time: %v\n",
			missingHours, issuesCount, timePerIssue, roundedTimePerIssue, extraTime)
		log.Printf("To track: %v", extraTime+(roundedTimePerIssue*float64(issuesCount)))

		for num, issue := range issues.Issues {
			timeToAdd := roundedTimePerIssue
			// Add extraTime to the first ticket
			if num == 0 {
				timeToAdd += extraTime
			}
			log.Printf("Tracking %v in #%v", timeToAdd, issue.ID)
			if !dryRun {
				go makeTimeEntry(host, apiKey, issue.ID, today, timeToAdd)
			}
		}
	}
}

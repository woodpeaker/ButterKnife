package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

const (
	activityId = 9
)

var (
	apiKey    string
	host      string
	dryRun    bool
	workHours float64
	today     string
	debug     bool
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
	flag.BoolVar(&debug, "debug", false, "Debug")
	flag.Float64Var(&workHours, "hours", 8.0, "Work `hours`")
	flag.StringVar(&today, "today", "", "Date to use as 'today'. Format is YYYY-MM-DD")
}

func apiGet(url string, container interface{}) (err error) {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		//err = errors.New(fmt.Sprintf("Response code %d", response.StatusCode))
		err = fmt.Errorf("Response code %d", response.StatusCode)
		return err
	}

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&container)
	return
}

func myIssues(host string, apiKey string) (issues issuesResult, err error) {
	url := "https://" + host + "/issues.json?assigned_to_id=me&limit=100&key=" + apiKey
	err = apiGet(url, &issues)
	return
}

func myTimeEntries(host string, apiKey string) (entries timeEntriesResult, err error) {
	url := "https://" + host + "/time_entries.json?user_id=me&sort=spent_on:desc&limit=100&key=" + apiKey
	err = apiGet(url, &entries)
	return
}

func makeTimeEntry(host string, apiKey string, issueID int, today string, timeToAdd float64) {
	url := "https://" + host + "/time_entries.json?key=" + apiKey

	var ir timeEntryRequest
	ir.TimeEntry = newTimeEntry{
		IssueID:    issueID,
		SpentOn:    today,
		Hours:      timeToAdd,
		ActivityID: activityId,
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
	response, err := client.Do(req)
	defer response.Body.Close()

	if err != nil {
		log.Fatal(err)
	}

	if debug {
		log.Println(response)
		body, _ := ioutil.ReadAll(response.Body)
		log.Printf("Body: %s", body)
	}
}

func trackTime(issues []issue, spentOn string, workHours float64, trackedTime float64) {
	var issuesCount = len(issues)
	var missingHours = workHours - trackedTime
	var timePerIssue = missingHours / float64(issuesCount)
	var roundedTimePerIssue = math.Floor(timePerIssue*10) / 10
	var extraTime = missingHours - (roundedTimePerIssue * float64(issuesCount))

	// min = 0.25
	// x = (missinHours / min)
	// rest = x % issuesCount
	// timePerIssue = x / issuesCount * min

	log.Printf("Missing hours: %v Issues: %v Time per issue: %v(rounded: %v) Extra time: %v To track: %v\n",
		missingHours,
		issuesCount,
		timePerIssue,
		roundedTimePerIssue,
		extraTime,
		extraTime+(roundedTimePerIssue*float64(issuesCount)))

	for num, issue := range issues {
		timeToAdd := roundedTimePerIssue
		// Add extraTime to the first ticket
		if num == 0 {
			timeToAdd += extraTime
		}
		log.Printf("Tracking %v in #%v", timeToAdd, issue.ID)
		if !dryRun {
			makeTimeEntry(host, apiKey, issue.ID, spentOn, timeToAdd)
		}
	}
}

func main() {
	var todayDate string
	var trackedTime = 0.0

	flag.Parse()

	if today == "" {
		todayDate = time.Now().Format("2006-01-02")
	} else {
		if todayDateButReallyDate, err := time.Parse("2006-01-02", today); err != nil {
			log.Fatalf("Invalid date format.")
		} else {
			todayDate = todayDateButReallyDate.Format("2006-01-02")
		}
	}

	log.Printf("todayDate is: %v\n", todayDate)

	if entries, err := myTimeEntries(host, apiKey); err == nil {
		for _, timeEntry := range entries.TimeEntries {
			if timeEntry.SpentOn == todayDate {
				trackedTime += timeEntry.Hours
			}
		}
	}

	log.Printf("Tracked todayDate: %v\n", trackedTime)

	if trackedTime < workHours {
		if issues, err := myIssues(host, apiKey); err == nil {
			trackTime(issues.Issues, todayDate, workHours, trackedTime)
		}
	}
}

package cal

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"

	// ics "github.com/JacopoD/golang-ical"
	ics "github.com/arran4/golang-ical"
)

func simpleGetRequest(url *string) (*string, bool) {

	var resp *http.Response
	var err error

	resp, err = http.Get(*url)

	if err != nil {
		return nil, true
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, true
	}

	var stringBody string = string(body)

	return &stringBody, false
}

func GetAllSubjects(url *string) (*map[string]int, *ics.Calendar) {

	r, err := simpleGetRequest(url)

	if err {
		// fmt.Println("Error")
		return nil, nil
	}

	cal, calErr := ics.ParseCalendar(strings.NewReader(*r))

	if calErr != nil {
		// fmt.Println("Error")
		return nil, nil
	}

	m := make(map[string]int)

	var url_prop, summary_prop *ics.IANAProperty

	for i := range cal.Components {
		switch event := cal.Components[i].(type) {
		case *ics.VEvent:
			url_prop = event.ComponentBase.GetProperty(ics.ComponentPropertyUrl)
			summary_prop = event.ComponentBase.GetProperty(ics.ComponentPropertySummary)

			if url_prop != nil {
				m[url_prop.Value] = 1

			} else if summary_prop != nil {
				m[summary_prop.Value] = 1
			}
		}
	}

	return &m, cal
}

func FilterCalendar(cal *ics.Calendar, oldMap *map[string]int, filter *[]string) *ics.Calendar {

	newMap := make(map[string]int)

	for _, element := range *filter {

		if val, ok := (*oldMap)[element]; ok {
			newMap[element] = val
		}
	}

	// set old map pointer to null so that GC will remove it
	oldMap = nil

	var url_prop, summary_prop *ics.IANAProperty

	var newComponents []ics.Component = []ics.Component{}

	for i := range (*cal).Components {
		switch event := (*cal).Components[i].(type) {
		case *ics.VEvent:

			url_prop = event.ComponentBase.GetProperty(ics.ComponentPropertyUrl)
			summary_prop = event.ComponentBase.GetProperty(ics.ComponentPropertySummary)

			if (url_prop != nil && newMap[url_prop.Value] == 1) || (summary_prop != nil && newMap[summary_prop.Value] == 1) {
				newComponents = append(newComponents, event)

			}
		default:
			newComponents = append(newComponents, event)
		}
	}

	(*cal).Components = newComponents

	return cal
}

func MergeRawCalendars(rawCals []*string) *string {
	var builder strings.Builder
	builder.WriteString("BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:USI Search\nX-WR-CALNAME:Custom USI Calendar - usicalendar.me\nX-WR-CALDESC:Custom USI Calendar - usicalendar.me\n")
	for _, cal := range rawCals {
		if cal == nil {
			continue
		}
		strippedRawCal := stripRawCal(cal)

		if strippedRawCal == nil {
			continue
		}

		builder.WriteString(*strippedRawCal)
	}
	builder.WriteString("END:VCALENDAR")
	var result string = builder.String()

	return &result
}

func stripRawCal(rawCal *string) *string {
	// Create a scanner to read the input
	scanner := bufio.NewScanner(strings.NewReader(*rawCal))

	// Initialize a flag to track if "BEGIN:VEVENT" is found
	foundBegin := false

	// Create a buffer to store the result
	var result strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if !foundBegin {
			if line == "BEGIN:VEVENT" {
				foundBegin = true
				result.WriteString(line)
				result.WriteString("\n")
			}
		} else {
			// Append the line to the result
			if line == "END:VCALENDAR" {
				break
			}
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error:", err)
		return nil
	}

	if !foundBegin {
		return nil
	}

	// Get the result as a string and print it
	outputStr := result.String()
	return &outputStr
}

func GetSubjCalFromIdx(idx *string) *string {
	var url string = "https://search.usi.ch/courses/" + *idx + "/*/schedules/ics"
	cal, err := simpleGetRequest(&url)
	if err {
		return nil
	}
	return cal
}

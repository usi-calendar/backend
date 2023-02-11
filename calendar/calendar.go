package cal

import (
	"io"
	"net/http"
	"strings"

	ics "github.com/JacopoD/golang-ical"
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

package routes

import (
	"strings"

	"github.com/gin-gonic/gin"

	cal "usicalendar/calendar"
	mongo "usicalendar/mongo"
)

const (
	ContentTypeJSON     = "application/json"
	ContentTypeHTML     = "text/html; charset=utf-8"
	ContentTypeText     = "text/plain; charset=utf-8"
	ContentTypeCalendar = "text/calendar"
)

func GetInfoFromUrl(c *gin.Context) {

	var url string = c.Query("url")

	if url == "" || !strings.HasPrefix(url, "https://search.usi.ch/") {
		c.Status(400)
		return
	}

	setAccessControlHeader(c)

	subjectsMap, _ := cal.GetAllSubjects(&url)

	if subjectsMap == nil {
		c.Status(400)
		return
	}

	var i int = 0
	subjects := make([]string, len(*subjectsMap))

	for key := range *subjectsMap {
		subjects[i] = strings.Clone(key)
		i++
	}
	subjectsNames := mongo.SubjIdToName(subjects)
	subjectsMap = nil

	var r string = "{\"courses\": ["
	i = 0
	var last int = len(subjects) - 1
	for _, subj := range subjects {
		r += `["` + subj + `","` + subjectsNames[i] + `"]`
		if i != last {
			r += ","
		}
		i++
	}
	r += "]}"

	c.Data(200, ContentTypeJSON, []byte(r))
}

func GetInfoFromId(c *gin.Context) {

	var idss string = c.Query("ids")

	if idss == "" {
		c.Status(400)
		return
	}

	ids := strings.Split(idss, "~")

	setAccessControlHeader(c)

	subjectsNames := mongo.SubjIdToName(ids)

	var r string = "{\"courses\": ["
	var last int = len(ids) - 1
	for i, el := range ids {
		r += `["` + el + `","` + subjectsNames[i] + `"]`
		if i != last {
			r += ","
		}
		i++
	}
	r += "]}"

	c.Data(200, ContentTypeJSON, []byte(r))
}

func GetShorten(c *gin.Context) {

	var url string = c.Query("url")
	var subjectsString string = c.Query("subjects")

	if url == "" || !strings.HasPrefix(url, "https://search.usi.ch/") || subjectsString == "" {
		c.Status(400)
		return
	}

	// c.Header("Access-Control-Allow-Origin", "*")
	setAccessControlHeader(c)

	subjects := strings.Split(subjectsString, "~")

	short := mongo.Shorten(&url, &subjects)

	if short == nil {
		c.Status(400)
		return
	}

	// var r string = "{\"shortened\":\"" + c.Request.Host + "/s/" + *short + "\"}"
	var r string = `{"shortened":"https://` + c.Request.Host + "/s/" + *short + `"}`

	c.Data(200, ContentTypeJSON, []byte(r))
}

func GetComplexShorten(c *gin.Context) {
	var url string = c.Query("url")
	var subjectsString string = c.Query("subjects")
	var extraSubjectsString string = c.Query("extra_subjects")
	var hasBaseCalendar string = c.Query("has_base_calendar")
	var hbcbool bool = false

	if hasBaseCalendar == "" || extraSubjectsString == "" {
		c.Status(400)
		return
	}

	if hasBaseCalendar == "true" {
		if url == "" || !strings.HasPrefix(url, "https://search.usi.ch/") || subjectsString == "" {
			c.Status(400)
			return
		}
		hbcbool = true
	} else {
		url = ""
		subjectsString = ""
	}

	setAccessControlHeader(c)

	subjects := strings.Split(subjectsString, "~")

	extraSubjects := strings.Split(extraSubjectsString, "~")

	short := mongo.ShortenComplex(hbcbool, &url, &subjects, &extraSubjects)

	if short == nil {
		c.Status(400)
		return
	}

	var r string = `{"shortened":"https://` + c.Request.Host + "/cs/" + *short + `"}`

	c.Data(200, ContentTypeJSON, []byte(r))
}

func GetShortened(c *gin.Context) {

	// c.Header("Access-Control-Allow-Origin", "*")
	setAccessControlHeader(c)

	var short string = c.Param("shortened")

	// fmt.Println(short)

	calendar := mongo.FromShortened(&short)

	if calendar == nil {
		c.Status(404)
		return
	}

	c.Data(200, ContentTypeCalendar, []byte(calendar.Serialize()))
}

func GetComplexShortened(c *gin.Context) {

	setAccessControlHeader(c)

	var short string = c.Param("shortened")

	calendar := mongo.FromComplexShortened(&short)

	if calendar == nil {
		c.Status(404)
		return
	}

	c.Data(200, ContentTypeCalendar, []byte(*calendar))
}

func GetCalendars(c *gin.Context) {
	setAccessControlHeader(c)
	var data *string = mongo.LatestCourses()

	if data == nil {
		c.Status(400)
		return
	}

	c.Data(200, ContentTypeJSON, []byte(*data))
}

func GetAllCourses(c *gin.Context) {
	setAccessControlHeader(c)
	var data *string = mongo.InfoAllCourses()

	if data == nil {
		c.Status(400)
		return
	}

	c.Data(200, ContentTypeJSON, []byte(*data))
}

func setAccessControlHeader(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
}

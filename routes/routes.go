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

func GetInfo(c *gin.Context) {

	var url string = c.Query("url")

	if url == "" || !strings.HasPrefix(url, "https://search.usi.ch/") {
		c.Status(400)
		return
	}

	setAccessControlHeader(c)

	subjects, _ := cal.GetAllSubjects(&url)

	if subjects == nil {
		c.Status(500)
		return
	}

	var r string = "{\"courses\": ["
	var i int = 0
	var last int = len(*subjects) - 1
	for key, _ := range *subjects {
		r += "\"" + key + "\""
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
	var subjectsString string = c.Query("courses")

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
	var r string = `{"shortened\":"https://` + c.Request.Host + "/s/" + *short + `"}`

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

func GetCalendars(c *gin.Context) {
	setAccessControlHeader(c)
	var data *string = mongo.LatestCourses()

	if data == nil {
		c.Status(500)
		return
	}

	c.Data(200, ContentTypeJSON, []byte(*data))
}

func setAccessControlHeader(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
}

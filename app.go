package main

import (
	"context"

	"github.com/gin-gonic/gin"

	routes "usicalendar/routes"

	mh "usicalendar/mongo_connection_handler"
)

func main() {

	defer mh.Cli.Disconnect(context.Background())

	// gin.SetMode(gin.ReleaseMode)
	gin.SetMode(gin.DebugMode)
	r := gin.Default()
	r.GET("/urlinfo", routes.GetInfoFromUrl)
	r.GET("/idinfo", routes.GetInfoFromId)
	r.GET("/shorten", routes.GetShorten)
	r.GET("/cshorten", routes.GetComplexShorten)
	r.GET("/s/:shortened", routes.GetShortened)
	r.GET("/cs/:shortened", routes.GetComplexShortened)
	r.GET("/courses", routes.GetCalendars)
	r.GET("/extcourses", routes.GetAllCourses)
	r.Run()
}

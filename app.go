package main

import (
	"context"

	"github.com/gin-gonic/gin"

	routes "usicalendar/routes"

	mongo "usicalendar/mongo"
)

func main() {

	defer mongo.Cli.Disconnect(context.TODO())

	gin.SetMode(gin.ReleaseMode)
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

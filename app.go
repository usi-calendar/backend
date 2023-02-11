package main

import (
	"github.com/gin-gonic/gin"

	routes "usicalendar/routes"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/info", routes.GetInfo)
	r.GET("/shorten", routes.GetShorten)
	r.GET("/s/:shortened", routes.GetShortened)
	r.GET("/courses", routes.GetCourses)
	r.Run()
}

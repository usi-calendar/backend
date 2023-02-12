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
	r.GET("/info", routes.GetInfo)
	r.GET("/shorten", routes.GetShorten)
	r.GET("/s/:shortened", routes.GetShortened)
	r.GET("/courses", routes.GetCalendars)
	r.Run()
}

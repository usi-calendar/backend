package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	mh "usicalendar/mongo_connection_handler"
	utils "usicalendar/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func FetchCourseCalendar(url *string) *string {
	var result mh.CourseCalendarCache
	err := mh.CourseCalendarCacheColl.FindOne(context.Background(), bson.D{{Key: "url", Value: *url}}).Decode(&result)

	if err != nil && err != mongo.ErrNoDocuments {
		return nil
	}

	if err == nil {
		updatedData, updated := updateCourseCache(&result)
		if updated {
			return updatedData
		}
		return &result.Data
	}

	rawCal, e := utils.SimpleGetRequest(url)

	if e {
		return nil
	}

	if !utils.IsCalendarValid(rawCal) {
		return nil
	}

	document := mh.CourseCalendarCache{
		ID:        primitive.NewObjectID(),
		Url:       *url,
		CID:       strings.Split((*url), "/")[5],
		DateAdded: time.Now().Unix(),
		Data:      *rawCal,
	}

	res, err := mh.CourseCalendarCacheColl.InsertOne(context.Background(), document)

	if err != nil || res.InsertedID == nil {
		fmt.Println(err)
		return nil
	}

	utils.Logger.Println("New course cache " + *url)

	return rawCal
}

func updateCourseCache(document *mh.CourseCalendarCache) (*string, bool) {

	if time.Now().Unix()-(*document).DateAdded < _MAX_AGE {
		return nil, false
	}

	rawCal, e := utils.SimpleGetRequest(&document.Url)

	if e {
		return nil, false
	}

	if !utils.IsCalendarValid(rawCal) {
		return nil, false
	}

	update := bson.D{{"$set", bson.D{{"data", *rawCal}, {"date_added", time.Now().Unix()}}}}

	res, err := mh.CourseCalendarCacheColl.UpdateByID(context.Background(), document.ID, update)

	if err != nil || res.ModifiedCount != 1 {
		fmt.Println(err)
		return nil, false
	}

	utils.Logger.Println("Updated course cache " + document.CID)

	return rawCal, true
}

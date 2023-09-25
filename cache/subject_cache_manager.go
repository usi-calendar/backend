package cache

import (
	"context"
	"fmt"
	"time"

	mh "usicalendar/mongo_connection_handler"
	utils "usicalendar/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func FetchSubjectCalendar(id *string) *string {
	// Check if cache document exists
	var result mh.SubjectCalendarCache
	err := mh.SubjectCalendarCacheColl.FindOne(context.Background(), bson.D{{Key: "id", Value: *id}}).Decode(&result)

	if err != nil && err != mongo.ErrNoDocuments {
		return nil
	}

	if err == nil {
		updatedData, updated := updateSubjectCache(&result)
		if updated {
			return updatedData
		}
		return &result.Data
	}

	url := "https://search.usi.ch/courses/" + *id + "/*/schedules/ics"

	rawCal, e := utils.SimpleGetRequest(&url)

	if e {
		return nil
	}

	document := mh.SubjectCalendarCache{
		ID:        primitive.NewObjectID(),
		SID:       *id,
		DateAdded: time.Now().Unix(),
		Data:      *rawCal,
	}

	res, err := mh.SubjectCalendarCacheColl.InsertOne(context.Background(), document)

	if err != nil || res.InsertedID == nil {
		fmt.Println(err)
		return nil
	}

	utils.Logger.Println("New subject cache " + *id)

	return rawCal
}

func updateSubjectCache(document *mh.SubjectCalendarCache) (*string, bool) {

	// if cache is too old, update
	if time.Now().Unix()-(*document).DateAdded < _MAX_AGE {
		return nil, false
	}

	url := "https://search.usi.ch/courses/" + (*document).SID + "/*/schedules/ics"

	rawCal, e := utils.SimpleGetRequest(&url)

	if e {
		return nil, false
	}

	update := bson.D{{"$set", bson.D{{"data", *rawCal}, {"date_added", time.Now().Unix()}}}}

	res, err := mh.SubjectCalendarCacheColl.UpdateByID(context.Background(), document.ID, update)

	if err != nil || res.ModifiedCount != 1 {
		fmt.Println(err)
		return nil, false
	}

	utils.Logger.Println("Updated cache for subject " + document.SID)

	return rawCal, true
}

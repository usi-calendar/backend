package mongo

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	cal "usicalendar/calendar"
	mh "usicalendar/mongo_connection_handler"
	"usicalendar/utils"

	ics "github.com/arran4/golang-ical"
)

var maxAttempts int = 200

func FromShortened(short *string) *ics.Calendar {
	var result *mh.ShortLink
	var err = mh.ShortLinksColl.FindOne(context.Background(), bson.D{{Key: "short_url", Value: *short}}).Decode(&result)

	if err != nil {
		return nil
	}

	subjects, calendar := cal.GetAllSubjects(&(*result).Url)

	calendar = cal.FilterCalendar(calendar, subjects, &(*result).Subjects)

	return calendar
}

func FromComplexShortened(short *string) *string {
	var result *mh.ComplexShortLink
	var err = mh.ComplexShortLinksColl.FindOne(context.Background(), bson.D{{Key: "short_url", Value: *short}}).Decode(&result)

	if err != nil {
		return nil
	}

	var baseCalendar *ics.Calendar
	var subjects *map[string]int
	var rawBaseCalendar string

	var l int = len(result.ExtraSubjects)

	if result.HasBaseCalendar {
		subjects, baseCalendar = cal.GetAllSubjects(&(*result).Url)
		baseCalendar = cal.FilterCalendar(baseCalendar, subjects, &(*result).BaseSubjects)
		rawBaseCalendar = baseCalendar.Serialize()
		l++
	}

	rawCals := make([]*string, l)

	if result.HasBaseCalendar {
		rawCals[l-1] = &rawBaseCalendar
	}

	for i, extra_subject := range result.ExtraSubjects {
		raw := cal.GetSubjCalFromIdx(&extra_subject)
		rawCals[i] = raw
	}

	return cal.MergeRawCalendars(rawCals)
}

func Shorten(url *string, filter *[]string) *string {

	if len(*filter) == 0 {
		return nil
	}

	subjects, _ := cal.GetAllSubjects(url)

	for _, f := range *filter {
		if (*subjects)[f] != 1 {
			return nil
		} else {
			if (*subjects)[f] > 1 {
				return nil
			}
			(*subjects)[f]++
		}
	}

	sort.Strings(*filter)

	// r = COL.find_one({"url":url, "courses" : f})

	var result *mh.ShortLink
	var err = mh.ShortLinksColl.FindOne(context.Background(),
		bson.D{{Key: "url", Value: *url}, {Key: "subjects", Value: *filter}}).Decode(&result)

	if err != nil && err != mongo.ErrNoDocuments {
		// SOMETHING IS WRONG IF THIS HAPPENS
		return nil
	}
	if err == nil {
		fmt.Println("Already shortened")
		return &result.Short_url
	}

	var i int
	var alphanum string
	for i = 0; i < maxAttempts+1; i++ {
		alphanum = utils.RandStringBytesMaskImprSrcSB(16)
		e := mh.ShortLinksColl.FindOne(context.Background(), bson.D{{Key: "short_url", Value: alphanum}}).Err()
		if e != nil {
			if e == mongo.ErrNoDocuments {
				break
			} else {
				return nil
			}
		}
	}

	if i == maxAttempts {
		return nil
	}

	res, err := mh.ShortLinksColl.InsertOne(context.Background(),
		bson.D{{Key: "url", Value: *url}, {Key: "subjects", Value: *filter}, {Key: "short_url", Value: alphanum}})

	// && res.InsertedID != nil USELESS check
	if err != nil || res.InsertedID == nil {
		return nil
	}

	return &alphanum
}

// hasBaseCalendar indicates whether the complex calendar is composed of:
// True: a combination of a base course + subjects
// False: just subjects
func ShortenComplex(hasBaseCalendar bool, url *string, baseFilter *[]string, extraSubjects *[]string) *string {
	if len(*extraSubjects) == 0 {
		return nil
	}

	sort.Strings(*extraSubjects)

	for i := 0; i < len(*extraSubjects)-1; i++ {
		if (*extraSubjects)[i] == (*extraSubjects)[i+1] {
			return nil
		}
	}

	if hasBaseCalendar {

		if len(*baseFilter) == 0 {
			return nil
		}

		subjects, _ := cal.GetAllSubjects(url)

		for _, f := range *baseFilter {
			if (*subjects)[f] != 1 {
				return nil
			} else {
				if (*subjects)[f] > 1 {
					return nil
				}
				(*subjects)[f]++
			}
		}

		sort.Strings(*baseFilter)

	}

	filter := bson.M{"subj_id": bson.M{"$in": *extraSubjects}}
	count, err := mh.SubjectsColl.CountDocuments(context.Background(), filter)

	if err != nil {
		return nil
	}
	if count != int64(len(*extraSubjects)) {
		return nil
	}

	var result *mh.ComplexShortLink
	err = mh.ComplexShortLinksColl.FindOne(context.Background(),
		bson.D{{Key: "has_base_calendar", Value: hasBaseCalendar},
			{Key: "url", Value: *url},
			{Key: "base_subjects", Value: *baseFilter},
			{Key: "extra_subjects", Value: *extraSubjects},
		}).Decode(&result)

	if err != nil && err != mongo.ErrNoDocuments {
		// SOMETHING IS WRONG IF THIS HAPPENS
		return nil
	}
	if err == nil {
		fmt.Println("Already shortened")
		return &result.Short_url
	}

	var i int
	var alphanum string
	for i = 0; i < maxAttempts+1; i++ {
		alphanum = utils.RandStringBytesMaskImprSrcSB(16)
		e := mh.ComplexShortLinksColl.FindOne(context.Background(), bson.D{{Key: "short_url", Value: alphanum}}).Err()
		if e != nil {
			if e == mongo.ErrNoDocuments {
				break
			} else {
				return nil
			}
		}
	}

	if i == maxAttempts {
		return nil
	}

	res, err := mh.ComplexShortLinksColl.InsertOne(context.Background(),
		bson.D{{Key: "has_base_calendar", Value: hasBaseCalendar},
			{Key: "url", Value: *url},
			{Key: "base_subjects", Value: *baseFilter},
			{Key: "extra_subjects", Value: *extraSubjects},
			{Key: "short_url", Value: alphanum}})

	// && res.InsertedID != nil USELESS check
	if err != nil || res.InsertedID == nil {
		return nil
	}

	return &alphanum
}

func LatestCourses() *string {
	// coursesColl := Db.Collection("courses")
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "date_added", Value: -1}}).SetLimit(1)

	cursor, err := mh.CoursesColl.Find(context.Background(), bson.D{}, findOptions)

	if err != nil {
		return nil
	}

	// There can only be one element in the cursor.
	var result mh.RawData
	for cursor.Next(context.Background()) {

		if err := cursor.Decode(&result); err != nil {

			return nil
		}
	}
	if err := cursor.Err(); err != nil {
		return nil
	}

	return &result.DataString
}

func SubjIdToName(ids []string) []string {
	// fmt.Println(ids)
	// filter := bson.M{"subj_id": bson.M{"$in": ids}}
	// cursor, err := SubjectsColl.Find(context.Background(), filter)

	pipeline := mongo.Pipeline{
		{
			{"$match", bson.M{"subj_id": bson.M{"$in": ids}}},
		},
		{
			{"$addFields", bson.M{"__order": bson.M{"$indexOfArray": []interface{}{ids, "$subj_id"}}}},
		},
		{
			{"$sort", bson.M{"__order": 1}},
		},
	}

	cursor, err := mh.SubjectsColl.Aggregate(context.Background(), pipeline)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	var result mh.Subject

	subjectNames := make([]string, len(ids))
	var i int = 0
	for cursor.Next(context.Background()) {
		if err := cursor.Decode(&result); err != nil {
			return nil
		}
		subjectNames[i] = strings.Clone(result.SubjName)
		i++
	}

	// fmt.Println(subjectNames)

	return subjectNames
}

func InfoCourse(id *string) (bool, *string, *string, []string) {
	var result mh.SubjectsAndCourse
	err := mh.SubjectsAndCoursesColl.FindOne(context.Background(), bson.D{{Key: "id", Value: *id}}).Decode(&result)

	if err != nil {
		return true, nil, nil, nil
	}

	return false, &result.CID, &result.CourseName, result.Subjects

}

func InfoAllCourses() *string {
	var result mh.RawData
	findOptions := options.FindOne()
	findOptions.SetSort(bson.D{{Key: "date_added", Value: -1}})

	err := mh.SubjectsAndCoursesRawColl.FindOne(context.Background(), bson.D{}, findOptions).Decode(&result)

	if err != nil {
		return nil
	}

	return &result.DataString
}

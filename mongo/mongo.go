package mongo

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	cal "usicalendar/calendar"
	"usicalendar/utils"

	// ics "github.com/JacopoD/golang-ical"
	ics "github.com/arran4/golang-ical"

	"github.com/joho/godotenv"
)

var Cli *mongo.Client = connection()

var Db *mongo.Database

var ShortLinksColl *mongo.Collection

var ComplexShortLinksColl *mongo.Collection

var SubjectsColl *mongo.Collection

var SubjectsAndCoursesColl *mongo.Collection

var SubjectsAndCoursesRawColl *mongo.Collection

const maxAttempts int = 2000

type ShortLink struct {
	ID        primitive.ObjectID `bson:"_id"`
	Url       string             `bson:"url,omitempty"`
	Subjects  []string           `bson:"subjects,omitempty"`
	Short_url string             `bson:"short_url,omitempty"`
}

type RawData struct {
	ID         primitive.ObjectID `bson:"_id"`
	DateAdded  primitive.DateTime `bson:"date_added,omitempty"`
	DataString string             `bson:"data,omitempty"`
}

type ComplexShortLink struct {
	ID              primitive.ObjectID `bson:"_id"`
	HasBaseCalendar bool               `bson:"has_base_calendar,omitempty"`
	Url             string             `bson:"url"`
	BaseSubjects    []string           `bson:"base_subjects"`
	ExtraSubjects   []string           `bson:"extra_subjects"`
	Short_url       string             `bson:"short_url,omitempty"`
}

type Subject struct {
	ID       primitive.ObjectID `bson:"_id"`
	SubjId   string             `bson:"subj_id,omitempty"`
	SubjName string             `bson:"subj_name,omitempty"`
}

type SubjectsAndCourse struct {
	ID         primitive.ObjectID `bson:"_id"`
	CID        string             `bson:"id,omitempty"`
	CourseName string             `bson:"course_name,omitempty"`
	Subjects   []string           `bson:"subjects,omitempty"`
}

func connection() *mongo.Client {

	// Load .env file
	err := godotenv.Load(".env")

	if err != nil {
		panic(err)
	}

	clientOptions := options.Client()
	clientOptions.ApplyURI(os.Getenv("MONGO_CONNECTION_STRING") + "&timeoutMS=5000")

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)

	if err != nil {
		panic(err)
	}

	// Check the connection
	if err := client.Ping(context.TODO(), nil); err != nil {
		panic(err)
	}

	Db = client.Database(os.Getenv("MONGO_DB_NAME"))

	ShortLinksColl = Db.Collection("short_links")

	ComplexShortLinksColl = Db.Collection("complex_short_links")

	SubjectsColl = Db.Collection("subjects")

	SubjectsAndCoursesColl = Db.Collection("subjects_and_courses")

	SubjectsAndCoursesRawColl = Db.Collection("subjects_and_courses_raw")

	fmt.Println("Connected to MongoDB!")

	return client
}

func FromShortened(short *string) *ics.Calendar {
	var result *ShortLink
	var err = ShortLinksColl.FindOne(context.Background(), bson.D{{Key: "short_url", Value: *short}}).Decode(&result)

	if err != nil {
		return nil
	}

	subjects, calendar := cal.GetAllSubjects(&(*result).Url)

	calendar = cal.FilterCalendar(calendar, subjects, &(*result).Subjects)

	return calendar
}

func FromComplexShortened(short *string) *string {
	var result *ComplexShortLink
	var err = ComplexShortLinksColl.FindOne(context.Background(), bson.D{{Key: "short_url", Value: *short}}).Decode(&result)

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

	var result *ShortLink
	var err = ShortLinksColl.FindOne(context.Background(),
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
		e := ShortLinksColl.FindOne(context.Background(), bson.D{{Key: "short_url", Value: alphanum}}).Err()
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

	res, err := ShortLinksColl.InsertOne(context.Background(),
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
func ShortenComplex(hasBaseCalendar bool, url *string, filter *[]string, extraSubjects *[]string) *string {
	if len(*extraSubjects) == 0 {
		return nil
	}

	if hasBaseCalendar {

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

	}

	var result *ComplexShortLink
	var err = ComplexShortLinksColl.FindOne(context.Background(),
		bson.D{{Key: "has_base_calendar", Value: hasBaseCalendar},
			{Key: "url", Value: *url},
			{Key: "base_subjects", Value: *filter},
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
		e := ComplexShortLinksColl.FindOne(context.Background(), bson.D{{Key: "short_url", Value: alphanum}}).Err()
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

	res, err := ComplexShortLinksColl.InsertOne(context.Background(),
		bson.D{{Key: "has_base_calendar", Value: hasBaseCalendar},
			{Key: "url", Value: *url},
			{Key: "base_subjects", Value: *filter},
			{Key: "extra_subjects", Value: *extraSubjects},
			{Key: "short_url", Value: alphanum}})

	// && res.InsertedID != nil USELESS check
	if err != nil || res.InsertedID == nil {
		return nil
	}

	return &alphanum
}

func LatestCourses() *string {
	coursesColl := Db.Collection("courses")
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "date_added", Value: -1}}).SetLimit(1)

	cursor, err := coursesColl.Find(context.Background(), bson.D{}, findOptions)

	if err != nil {
		return nil
	}

	// There can only be one element in the cursor.
	var result RawData
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
	fmt.Println(ids)
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

	cursor, err := SubjectsColl.Aggregate(context.Background(), pipeline)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	var result Subject

	subjectNames := make([]string, len(ids))
	var i int = 0
	for cursor.Next(context.Background()) {
		if err := cursor.Decode(&result); err != nil {
			return nil
		}
		subjectNames[i] = strings.Clone(result.SubjName)
		i++
	}

	fmt.Println(subjectNames)

	return subjectNames
}

func InfoCourse(id *string) (bool, *string, *string, []string) {
	var result SubjectsAndCourse
	err := SubjectsAndCoursesColl.FindOne(context.Background(), bson.D{{Key: "id", Value: *id}}).Decode(&result)

	if err != nil {
		return true, nil, nil, nil
	}

	return false, &result.CID, &result.CourseName, result.Subjects

}

func InfoAllCourses() *string {
	var result RawData
	findOptions := options.FindOne()
	findOptions.SetSort(bson.D{{Key: "date_added", Value: -1}})

	err := SubjectsAndCoursesRawColl.FindOne(context.Background(), bson.D{}, findOptions).Decode(&result)

	if err != nil {
		return nil
	}

	return &result.DataString
}

package mongo_connection_handler

import (
	"context"
	"os"
	"usicalendar/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Cli *mongo.Client = connection()

var Db *mongo.Database

var ShortLinksColl *mongo.Collection

var ComplexShortLinksColl *mongo.Collection

var SubjectsColl *mongo.Collection

var SubjectsAndCoursesColl *mongo.Collection

var SubjectsAndCoursesRawColl *mongo.Collection

var CourseCalendarCacheColl *mongo.Collection

var SubjectCalendarCacheColl *mongo.Collection

var CoursesColl *mongo.Collection

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

type CourseCalendarCache struct {
	ID        primitive.ObjectID `bson:"_id"`
	Url       string             `bson:"url,omitempty"`
	CID       string             `bson:"id,omitempty"`
	Data      string             `bson:"data,omitempty"`
	DateAdded int64              `bson:"date_added,omitempty"`
}

type SubjectCalendarCache struct {
	ID        primitive.ObjectID `bson:"_id"`
	SID       string             `bson:"id,omitempty"`
	Data      string             `bson:"data,omitempty"`
	DateAdded int64              `bson:"date_added,omitempty"`
}

func connection() *mongo.Client {

	// UNCOMMENT FOR DEBUGGING WITH .ENV FILE
	// ######################################
	// Load .env file

	// err := godotenv.Load(".env")

	// if err != nil {
	// 	panic(err)
	// }

	// ######################################

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

	CourseCalendarCacheColl = Db.Collection("course_calendar_cache")

	SubjectCalendarCacheColl = Db.Collection("subject_calendar_cache")

	CoursesColl = Db.Collection("courses")

	utils.Logger.Println("Connected to MongoDB!")

	return client
}

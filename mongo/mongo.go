package mongo

import (
	"context"
	"fmt"
	"os"
	"sort"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	cal "usicalendar/calendar"
	"usicalendar/utils"

	ics "github.com/JacopoD/golang-ical"

	"github.com/joho/godotenv"
)

var Cli *mongo.Client = connection()

var Db *mongo.Database

var ShortLinksColl *mongo.Collection

const maxAttempts int = 2000

type ShortLink struct {
	ID        primitive.ObjectID `bson:"_id"`
	Url       string             `bson:"url,omitempty"`
	Courses   []string           `bson:"courses,omitempty"`
	Short_url string             `bson:"short_url,omitempty"`
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

	Db = client.Database("usi-calendar-development")

	ShortLinksColl = Db.Collection("short_links")

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

	calendar = cal.FilterCalendar(calendar, subjects, &(*result).Courses)

	return calendar
}

func Shorten(url *string, filter *[]string) *string {

	if len(*filter) == 0 {
		return nil
	}

	subjects, _ := cal.GetAllSubjects(url)

	for _, f := range *filter {
		if (*subjects)[f] != 1 {
			return nil
		}
	}

	sort.Strings(*filter)

	// r = COL.find_one({"url":url, "courses" : f})

	var result *ShortLink
	var err = ShortLinksColl.FindOne(context.Background(),
		bson.D{{Key: "url", Value: *url}, {Key: "courses", Value: *filter}}).Decode(&result)

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
		bson.D{{Key: "url", Value: *url}, {Key: "courses", Value: *filter}, {Key: "short_url", Value: alphanum}})

	// && res.InsertedID != nil USELESS check
	if err != nil && res.InsertedID != nil {
		return nil
	}

	return &alphanum
}

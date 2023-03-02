package main

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"go-crypto-hft/codec"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"reflect"
	"time"
)

type Quote struct {
	ID     primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Ticker string             `json:"ticker" bson:"ticker"`
	Time   primitive.DateTime `json:"time" bson:"time"`
	Bid    decimal.Decimal    `json:"bid" bson:"bid"`
	Ask    decimal.Decimal    `json:"ask" bson:"ask"`
}

type MongoInstance struct {
	Client *mongo.Client
	DB     *mongo.Database
}

var mi = MongoInstance{}

const dbName = "crypto-hft"
const mongoURI = "mongodb://localhost:27017/" + dbName

func Connect() error {
	opts := options.Client().SetRegistry(
		bson.NewRegistryBuilder().RegisterCodec(
			reflect.TypeOf(decimal.Decimal{}),
			&codec.DecimalCodec{},
		).Build()).ApplyURI(mongoURI)
	client, err := mongo.NewClient(opts)
	if err != nil {
		log.Fatal(err)
		return err
	}
	mi.Client = client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
		return err
	}
	mi.DB = client.Database(dbName)
	return nil
}

const dateFormat = "2006-01-02T15:04:05 -0700"

func main() {
	if err := Connect(); err != nil {
		log.Fatal(err)
		return
	}
	app := fiber.New()
	app.Get("/quotes/:ticker", func(ctx *fiber.Ctx) error {
		str := ctx.Params("startTime", time.Now().Add(-(time.Hour * 3)).Format(time.RFC3339))
		startTime, err := time.Parse(time.RFC3339, str)
		if err != nil {
			log.Printf(err.Error())
			return err
		}
		endTime, err := time.Parse(time.RFC3339, ctx.Params("endTime", time.Now().Format(time.RFC3339)))
		if err != nil {
			log.Printf("Invalid endTime received: " + ctx.Params("endTime"))
			return err
		}
		ticker := ctx.Params("ticker")

		filter := bson.D{
			{Key: "$and",
				Value: bson.A{
					bson.M{"ticker": ticker},
					bson.M{"time": bson.M{"$gt": primitive.NewDateTimeFromTime(startTime),
						"$lt": primitive.NewDateTimeFromTime(endTime)}},
				},
			},
		}

		quotes := make([]Quote, 0)
		cursor, err := mi.DB.Collection("quotes").Find(ctx.Context(), filter)
		if err != nil {
			log.Printf(err.Error())
			_ = ctx.Status(500).SendString(err.Error())
			return err
		}
		err = cursor.All(ctx.Context(), &quotes)
		if err != nil {
			log.Printf(err.Error())
			_ = ctx.Status(500).SendString(err.Error())
			return err
		}
		err = ctx.Status(200).JSON(quotes)
		if err != nil {
			log.Fatal(err)
			return err
		}
		return nil
	})
	app.Post("quotes", func(ctx *fiber.Ctx) error {
		var quote = Quote{}
		if err := ctx.BodyParser(&quote); err != nil {
			log.Printf(err.Error())
			_ = ctx.Status(500).SendString(err.Error())
			return err
		}
		insertResult, err := mi.DB.Collection("quotes").InsertOne(ctx.Context(), &quote)
		if err != nil {
			log.Printf(err.Error())
			_ = ctx.Status(500).SendString(err.Error())
			return err
		}
		quote.ID = insertResult.InsertedID.(primitive.ObjectID)
		if err := ctx.Status(200).JSON(quote); err != nil {
			log.Fatal(err)
			return err
		}
		return nil
	})
	log.Fatal(app.Listen(":8080"))
}

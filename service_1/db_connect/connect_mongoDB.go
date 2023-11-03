package db_connect

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoCollection *mongo.Collection

func InitializeMongoCollection() error {
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Printf("Ошибка подключения к MongoDB: %v", err)
		return err
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Printf("Ошибка проверки связи MongoDB: %v", err)
		return err
	}

	database := client.Database("grpc-golang")
	collection := database.Collection("bank")

	mongoCollection = collection
	return nil
}

func GetMongoCollection() (*mongo.Collection, error) {
	if mongoCollection == nil {
		return nil, fmt.Errorf("MongoDB коллекция не инициализирована")
	}
	return mongoCollection, nil
}

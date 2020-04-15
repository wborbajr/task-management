package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// Path to the AWS CA file
	// caFilePath = "rds-combined-ca-bundle.pem"

	// Timeout operations after N seconds
	connectTimeout           = 5
	queryTimeout             = 30
	connectionStringTemplate = "mongodb://%s:%s@%s"
)

// GetConnection Retrieves a client to the DocumentDB
func getConnection() (*mongo.Client, context.Context, context.CancelFunc) {
	username := os.Getenv("MONGODB_USERNAME")
	password := os.Getenv("MONGODB_PASSWORD")
	clusterEndpoint := os.Getenv("MONGODB_ENDPOINT")

	connectionURI := fmt.Sprintf(connectionStringTemplate, username, password, clusterEndpoint)

	log.Print(connectionURI)
	client, err := mongo.NewClient(options.Client().ApplyURI(connectionURI))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout*time.Second)

	err = client.Connect(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to cluster: %v", err)
	}

	// Force a connection to verify our connection string
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping cluster: %v", err)
	}

	fmt.Println("Connected to DocumentDB!")
	return client, ctx, cancel
}

// GetAllTasks Retrives all tasks from the db
func GetAllTasks() ([]*Task, error) {
	var tasks []*Task

	client, ctx, cancel := getConnection()
	defer cancel()
	defer client.Disconnect(ctx)
	db := client.Database("tasks")
	collection := db.Collection("tasks")
	cursor, err := collection.Find(ctx, bson.D{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	err = cursor.All(ctx, &tasks)
	if err != nil {
		log.Fatalf("Failed marshalling %v", err)
		return nil, err
	}
	return tasks, nil
}

// GetTaskByID Retrives a task by its id from the db
func GetTaskByID(id primitive.ObjectID) (*Task, error) {
	var task *Task

	client, ctx, cancel := getConnection()
	defer cancel()
	defer client.Disconnect(ctx)
	db := client.Database("tasks")
	collection := db.Collection("tasks")
	result := collection.FindOne(ctx, bson.D{})
	if result == nil {
		return nil, errors.New("Could not find a Task")
	}
	err := result.Decode(&task)

	if err != nil {
		log.Fatalf("Failed marshalling %v", err)
		return nil, err
	}
	log.Printf("Tasks: %v", task)
	return task, nil
}

//Create creating a task in a mongo or document db
func Create(task *Task) (primitive.ObjectID, error) {
	client, ctx, cancel := getConnection()
	defer cancel()
	defer client.Disconnect(ctx)
	task.ID = primitive.NewObjectID()

	result, err := client.Database("tasks").Collection("tasks").InsertOne(ctx, task)
	if err != nil {
		log.Fatalf("Could not create Task: %v", err)
		return primitive.NilObjectID, err
	}
	oid := result.InsertedID.(primitive.ObjectID)
	return oid, nil
}

//Update updating an existing task in a mongo or document db
func Update(task *Task) (*Task, error) {
	var updatedTask *Task
	client, ctx, cancel := getConnection()
	defer cancel()
	defer client.Disconnect(ctx)

	update := bson.M{
		"$set": task,
	}

	upsert := true
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		Upsert:         &upsert,
		ReturnDocument: &after,
	}

	err := client.Database("tasks").Collection("tasks").FindOneAndUpdate(ctx, bson.M{"_id": task.ID}, update, &opt).Decode(&updatedTask)
	if err != nil {
		log.Fatalf("Could not save Task: %v", err)
		return nil, err
	}
	return updatedTask, nil
}

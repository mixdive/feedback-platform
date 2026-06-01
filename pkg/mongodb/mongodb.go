package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

//var db *MongoDB

type MongoDB struct {
	ConnectionString string
	DBName           string
	Client           *mongo.Client
	isLoggingEnabled bool
}

func New(connectionString string, dbName string, loggingEnabled bool) *MongoDB {
	return &MongoDB{
		ConnectionString: connectionString,
		DBName:           dbName,
		Client:           nil,
		isLoggingEnabled: loggingEnabled,
	}
}

func (m *MongoDB) logMongo() {
	if !m.isLoggingEnabled {
		return
	}

	callers := make([]string, 0)

	for i := 3; i >= 1; i-- {
		if pc, _, _, ok := runtime.Caller(i); !ok {
			fmt.Println("Unable to retrieve caller information")
			return
		} else {
			cf := runtime.FuncForPC(pc).Name()
			cf = runtime.FuncForPC(pc).Name()
			cf = strings.Replace(cf, "internal/", "", -1)
			cf = strings.Replace(cf, "mongodb.(*MongoDB).", "", -1)
			cf = strings.Replace(cf, ".(*DataOperations)", "", -1)
			callers = append(callers, cf)
		}
	}

	finalLogMessage := ""
	for index, caller := range callers {
		finalLogMessage += caller
		if index != len(callers)-1 {
			finalLogMessage += " -- >"
		}
	}

	log.Println(fmt.Sprintf("MONGODB REQUEST: %s", finalLogMessage))
}

func (m *MongoDB) getClient() (*mongo.Client, error) {
	if m.Client != nil {
		// Ping the server to verify the connection is still alive
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := m.Client.Ping(ctx, nil)
		if err == nil {
			// If there is no error, return the existing client
			return m.Client, nil
		}
		// If ping failed, close the client and try to reconnect
		m.Client.Disconnect(context.Background())
		m.Client = nil
	}

	m.logMongo()

	var err error
	clientOptions := options.Client().ApplyURI(m.ConnectionString).
		SetServerSelectionTimeout(5 * time.Second). // Timeout for selecting a server
		SetConnectTimeout(10 * time.Second).        // Timeout for establishing a connection
		SetMaxPoolSize(100).                        // Maximum number of connections in the pool
		SetMinPoolSize(5).                          // Minimum number of connections in the pool
		SetMaxConnIdleTime(5 * time.Minute)         // Maximum time that a connection can be idle

	// Try to connect with retries
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		m.Client, err = mongo.Connect(clientOptions)
		cancel()

		if err == nil {
			// Verify the connection
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			err = m.Client.Ping(ctx, nil)
			cancel()

			if err == nil {
				return m.Client, nil
			}
		}

		// If we get here, there was an error
		if m.Client != nil {
			m.Client.Disconnect(context.Background())
			m.Client = nil
		}

		// Wait before retrying
		if i < maxRetries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	return nil, fmt.Errorf("failed to connect to MongoDB after %d attempts: %v", maxRetries, err)
}

func (m *MongoDB) Close() {
	if m.Client == nil {
		return
	}

	err := m.Client.Disconnect(context.TODO())
	if err != nil {
		log.Println(err)
	}

	m.Client = nil
}

func GetOneById[T any](db *MongoDB, collectionName string, id string) (*T, error) {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return nil, err
	}

	collection := client.Database(db.DBName).Collection(collectionName)

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	//defer client.Disconnect(ctx)

	filter := bson.D{{"_id", id}}

	var result *T
	err = collection.FindOne(ctx, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return result, err
}

func InsertOne[T any](db *MongoDB, collectionName string, record T) error {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return err
	}

	collection := client.Database(db.DBName).Collection(collectionName)

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	_, err = collection.InsertOne(ctx, record)
	return err
}

func UpdateOne[T any](db *MongoDB, collectionName string, id string, record *T) error {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return err
	}

	collection := client.Database(db.DBName).Collection(collectionName)
	filter := bson.D{{"_id", id}}
	update := bson.D{{"$set", record}}
	opts := options.UpdateOne() //.SetUpsert(true)

	ctx, _ := context.WithTimeout(context.Background(), 15*time.Second)

	_, err = collection.UpdateOne(ctx, filter, update, opts) //result
	if err != nil {
		return err
	}

	return nil
}

func IncrementValue(db *MongoDB, collectionName string, id string, fieldName string, value int) error {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return err
	}

	update := bson.M{"$inc": bson.M{fieldName: value}}

	collection := client.Database(db.DBName).Collection(collectionName)

	// Update one document
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": id}, update)
	if err != nil {
		return err
	}

	//log.Println(result)
	return err
}

func SetValue(db *MongoDB, collectionName string, id string, fieldName string, value any) error {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return err
	}

	update := bson.M{"$set": bson.M{fieldName: value}}

	collection := client.Database(db.DBName).Collection(collectionName)

	// Update one document
	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": id}, update)
	if err != nil {
		return err
	}

	//log.Println(result)
	return err
}

// UnsetValue removes a single field from one document by ID. Sibling of
// SetValue for $unset-style migrations that retire a renamed BSON key
// after copying its contents to a new one.
func UnsetValue(db *MongoDB, collectionName string, id string, fieldName string) error {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return err
	}

	update := bson.M{"$unset": bson.M{fieldName: ""}}

	collection := client.Database(db.DBName).Collection(collectionName)

	_, err = collection.UpdateOne(context.TODO(), bson.M{"_id": id}, update)
	return err
}

func UpsertOne[T any](db *MongoDB, collectionName string, id string, record *T) error {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return err
	}

	collection := client.Database(db.DBName).Collection(collectionName)
	filter := bson.D{{"_id", id}}
	update := bson.D{{"$set", record}}
	opts := options.UpdateOne().SetUpsert(true)

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	//defer client.Disconnect(ctx)

	_, err = collection.UpdateOne(ctx, filter, update, opts) //result
	if err != nil {
		return err
	}

	//fmt.Printf("Number of documents updated: %v\n", result.ModifiedCount)
	//fmt.Printf("Number of documents upserted: %v\n", result.UpsertedCount)

	return nil
}

func BulkWrite(db *MongoDB, collectionName string, writeModels []mongo.WriteModel) (int, error) {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return 0, err
	}

	bulkWriteOptions := options.BulkWrite().SetOrdered(false)

	collection := client.Database(db.DBName).Collection(collectionName)

	numberOfDuplicates := 0

	_, err = collection.BulkWrite(context.TODO(), writeModels, bulkWriteOptions)
	if err != nil {
		if bulkWriteException, ok := err.(mongo.BulkWriteException); ok {
			for _, writeError := range bulkWriteException.WriteErrors {
				if writeError.Code == 11000 { // Duplicate key error code
					numberOfDuplicates++
					//No problem. Review is already in the database
					//fmt.Printf("Duplicate key error: %v\n", writeError.Message)
				} else {
					return numberOfDuplicates, err
				}
			}
		} else {
			return numberOfDuplicates, err
		}
	}

	//fmt.Printf("Inserted %d documents\n", result.InsertedCount)

	return numberOfDuplicates, nil
}

func BulkUpdate(db *MongoDB, collectionName string, updateModels []mongo.WriteModel) (int, error) {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return 0, err
	}

	bulkWriteOptions := options.BulkWrite().SetOrdered(false)

	collection := client.Database(db.DBName).Collection(collectionName)

	numberOfFailures := 0

	_, err = collection.BulkWrite(context.TODO(), updateModels, bulkWriteOptions)
	if err != nil {
		if bulkWriteException, ok := err.(mongo.BulkWriteException); ok {
			for _, _ = range bulkWriteException.WriteErrors {
				// Handle specific error codes if necessary
				numberOfFailures++
				// Uncomment below line for debug logging
				// fmt.Printf("Write error: %v\n", writeError.Message)
			}
		} else {
			return numberOfFailures, err
		}
	}

	numberOfUpdates := len(updateModels) - numberOfFailures

	return numberOfUpdates, nil
}

func Query[T any](db *MongoDB, collectionName string, filter interface{}, opts *options.FindOptionsBuilder) ([]T, error) {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return nil, err
	}

	collection := client.Database(db.DBName).Collection(collectionName)

	ctx, _ := context.WithTimeout(context.Background(), 60*time.Second)
	//defer cancel()
	//defer client.Disconnect(ctx)

	//filter := bson.D{{"syncstatus.lastreleasecheckdate", nil}}
	//opts := options.Find().SetSort(bson.D{{"credate", -1}})

	if cursor, err := collection.Find(ctx, filter, opts); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}

		return nil, err
	} else {
		var result []T
		if err = cursor.All(ctx, &result); err != nil {
			return nil, err
		}

		if result == nil {
			return []T{}, nil
		}

		return result, nil
	}
}

func QueryOne[T any](db *MongoDB, collectionName string, filter interface{}, opts *options.FindOptionsBuilder) (*T, error) {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return nil, err
	}

	collection := client.Database(db.DBName).Collection(collectionName)

	ctx, _ := context.WithTimeout(context.Background(), 60*time.Second)

	if cursor, err := collection.Find(ctx, filter, opts); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		return nil, err
	} else {
		var result []T
		if err = cursor.All(ctx, &result); err != nil {
			return nil, err
		}

		if len(result) == 1 {
			return &result[0], nil
		}

		return nil, nil
	}
}

func GetAll[T any](db *MongoDB, collectionName string) ([]T, error) {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return nil, err
	}

	collection := client.Database(db.DBName).Collection(collectionName)
	filter := bson.D{}

	ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
	//defer cancel()
	//defer client.Disconnect(ctx)

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	var result []T
	if err = cursor.All(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func DeleteOne(db *MongoDB, collectionName string, id string) error {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return err
	}

	collection := client.Database(db.DBName).Collection(collectionName)

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	//defer cancel()
	//defer client.Disconnect(ctx)

	filter := bson.D{{"_id", id}}

	_, err = collection.DeleteOne(ctx, filter)
	return err
}

func DeleteAll(db *MongoDB, collectionName string, filter interface{}) error {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return err
	}

	collection := client.Database(db.DBName).Collection(collectionName)

	ctx, _ := context.WithTimeout(context.Background(), 100*time.Second)
	_, err = collection.DeleteMany(ctx, filter)
	return err
}

func Count(db *MongoDB, collectionName string, filter interface{}, opts *options.CountOptionsBuilder) (int64, error) {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return -1, err
	}

	collection := client.Database(db.DBName).Collection(collectionName)

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	if count, err := collection.CountDocuments(ctx, filter, opts); err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, nil
		}

		return -1, err
	} else {
		return count, nil
	}
}

// FindOneAndUpdate runs an atomic findOneAndUpdate against the given
// collection. Returns the document AFTER the update by default — pass
// options to switch to before-image semantics if needed. Returns
// (nil, nil) when the filter matches nothing (the collection method
// returns mongo.ErrNoDocuments which we translate to a nil result).
//
// This is the foundation for cluster-safe atomic claims: the AI
// analysis worker uses this to claim entries for processing, so two
// Cloud Run instances polling at the same instant cannot pick up the
// same entry.
func FindOneAndUpdate[T any](db *MongoDB, collectionName string, filter interface{}, update interface{}, opts *options.FindOneAndUpdateOptionsBuilder) (*T, error) {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return nil, err
	}
	collection := client.Database(db.DBName).Collection(collectionName)
	ctx, _ := context.WithTimeout(context.Background(), 60*time.Second)

	if opts == nil {
		opts = options.FindOneAndUpdate().SetReturnDocument(options.After)
	}

	res := collection.FindOneAndUpdate(ctx, filter, update, opts)
	if err := res.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}

	var out T
	if err := res.Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func Aggregate[T any](db *MongoDB, collectionName string, pipeline interface{}) ([]T, error) {
	db.logMongo()
	client, err := db.getClient()
	if err != nil {
		return nil, err
	}

	collection := client.Database(db.DBName).Collection(collectionName)
	ctx, _ := context.WithTimeout(context.Background(), 60*time.Second)

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	var results []T
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	if results == nil {
		return []T{}, nil
	}

	return results, nil
}

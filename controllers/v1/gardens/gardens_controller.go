package gardens_controllers

import (
	"context"
	"fmt"
	"log"
	"time"

	completed_tasks_controllers "github.com/ice-cream-backend/controllers/v1/completed_tasks"
	garden_categories_controllers "github.com/ice-cream-backend/controllers/v1/garden_categories"
	rules_controllers "github.com/ice-cream-backend/controllers/v1/rules"
	mongo_connection "github.com/ice-cream-backend/database"
	completed_tasks_models "github.com/ice-cream-backend/models/v1/completed_tasks"
	gardens_models "github.com/ice-cream-backend/models/v1/gardens"
	rules_models "github.com/ice-cream-backend/models/v1/rules"
	"github.com/ice-cream-backend/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateGardens(createdGardensPost gardens_models.GardenForMongo) (*mongo.InsertOneResult, error) {
	start := utils.StartPerformanceTest()
	ctx, ctxCancel := mongo_connection.ContextForMongo()
	client := mongo_connection.MongoConnection(ctx)

	defer client.Disconnect(ctx)
	defer ctxCancel()

	collection := mongo_connection.MongoCollection(client, "gardens")

	createdGardensPost.CreatedDate = time.Now()
	createdGardensPost.LastUpdate = time.Now()

	fmt.Printf("check create garden: %+v", createdGardensPost)

	res, insertErr := collection.InsertOne(ctx, createdGardensPost)

	if insertErr != nil {
		log.Println("Error creating new createGardens:", insertErr)
	}

	utils.StopPerformanceTest(start, "Successful create garden (controller)")
	return res, insertErr
}

func GetGardensByGardenId(createGardenId interface{}) (gardens_models.Gardens, error) {
	start := utils.StartPerformanceTest()
	ctx, ctxCancel := mongo_connection.ContextForMongo()
	client := mongo_connection.MongoConnection(ctx)

	defer client.Disconnect(ctx)
	defer ctxCancel()

	collection := mongo_connection.MongoCollection(client, "gardens")

	var result gardens_models.Gardens
	err := collection.FindOne(ctx, bson.D{
		primitive.E{Key: "_id", Value: createGardenId},
	},
	).Decode(&result)
	
	if err != nil {
		log.Println("err in findOne:", err)
		utils.StopPerformanceTest(start, "Unsuccessful getting gardensByGardenId (controller)")
		return result, err
	} else {
		gardenCategory, gardenCategoryErr := garden_categories_controllers.GetGardenCategoryByGardenCategoryId(result.GardenCategoryId)

		if gardenCategoryErr != nil {
			log.Println("Error getting gardenCategory for garden:", gardenCategoryErr)
		}

		result.GardenCategory = gardenCategory

		utils.StopPerformanceTest(start, "Successful get gardensByGardenId (controller)")
		return result, gardenCategoryErr
	}
}

func GetPopulatedGardenByGardenId(gardenId interface{}, date string) (gardens_models.GardensFullyPopulated, error) {
	garden, err := GetGardensByGardenId(gardenId)

	var populatedGarden gardens_models.GardensFullyPopulated

	if err != nil {
		return populatedGarden, err
	}

	rules := rules_controllers.GetRulesByGardenId(gardenId)

	var ruleIds []interface{}
	for _, rule := range rules {
		ruleIds = append(ruleIds, rule.ID)
	}

	populatedGarden.Garden = garden

	if len(ruleIds) == 0 {
		populatedGarden.Rules = []rules_models.Rules{}
		populatedGarden.CompletedTasks = []completed_tasks_models.CompletedTasks{}
	} else{
		populatedGarden.Rules = rules

		goDate := utils.ConvertAPIStringToDate(date)

		completedTasks := completed_tasks_controllers.GetCompletedTasksByRuleIdWithDate(ruleIds, goDate)
		populatedGarden.CompletedTasks = completedTasks
	}

	return populatedGarden, nil
}

func GetPopulatedGardenByGardenIdWithStartAndEndDate(gardenId interface{}, startDate string, endDate string) (gardens_models.GardensFullyPopulated, error) {
	garden, err := GetGardensByGardenId(gardenId)

	var populatedGarden gardens_models.GardensFullyPopulated

	if err != nil {
		return populatedGarden, err
	}

	rules := rules_controllers.GetRulesByGardenId(gardenId)

	var ruleIds []interface{}
	for _, rule := range rules {
		ruleIds = append(ruleIds, rule.ID)
	}

	populatedGarden.Garden = garden

	if len(ruleIds) == 0 {
		populatedGarden.Rules = []rules_models.Rules{}
		populatedGarden.CompletedTasks = []completed_tasks_models.CompletedTasks{}
	} else{
		populatedGarden.Rules = rules

		goStartDate := utils.ConvertAPIStringToDate(startDate)
		goEndDate := utils.ConvertAPIStringToDate(endDate)

		completedTasks := completed_tasks_controllers.GetCompletedTasksByRuleIdWithStartAndEndDate(ruleIds, goStartDate, goEndDate)
		populatedGarden.CompletedTasks = completedTasks
	}

	return populatedGarden, nil
}

func GetGardensByUserId(fireBaseUserId interface{}) []gardens_models.Gardens {
	ctx, ctxCancel := mongo_connection.ContextForMongo()
	client := mongo_connection.MongoConnection(ctx)

	defer client.Disconnect(ctx)
	defer ctxCancel()

	collection := mongo_connection.MongoCollection(client, "gardens")

	var results []gardens_models.Gardens
	query := bson.D{
		primitive.E{Key: "fireBaseUserId", Value: fireBaseUserId},
	}
	cursor, err := collection.Find(ctx, query)
	if err != nil {
		log.Println(err)
	}

	cursorErr := cursor.All(context.TODO(), &results)

	if cursorErr != nil {
		log.Println(cursorErr)
	}

	gardenCategories, gardenCategoryErr := garden_categories_controllers.GetGardenCategories()

	if gardenCategoryErr != nil {
		log.Println("Error getting garden categories to populate Gardens by fireBaseUserId")
	}

	var populatedGardens []gardens_models.Gardens
	for _, garden := range results {
		for _, gardenCategory := range gardenCategories {
			if gardenCategory.ID == garden.GardenCategoryId {
				garden.GardenCategory = gardenCategory
				break
			}
		}
		populatedGardens = append(populatedGardens, garden)
	}

	return populatedGardens
}

func UpdateGardenByGardenId(gardenId interface{}, garden gardens_models.Gardens) (*mongo.UpdateResult, error) {
	ctx, ctxCancel := mongo_connection.ContextForMongo()
	client := mongo_connection.MongoConnection(ctx)

	defer client.Disconnect(ctx)
	defer ctxCancel()

	collection := mongo_connection.MongoCollection(client, "gardens")

	updatedGarden := bson.M{
		"$set": bson.M{
			"name": garden.Name,
			"description": garden.Description,
			"gardenCategoryId": garden.GardenCategoryId,
			"lastUpdate": time.Now(),
		},
	}

	result, updateErr := collection.UpdateByID(ctx, gardenId, updatedGarden)

	return result, updateErr
}

func DeleteGardenByGardenId(gardenId interface{}) (*mongo.DeleteResult, error) {
	ctx, ctxCancel := mongo_connection.ContextForMongo()
	client := mongo_connection.MongoConnection(ctx)

	defer client.Disconnect(ctx)
	defer ctxCancel()

	collection := mongo_connection.MongoCollection(client, "gardens")

	query := bson.D{
		primitive.E{Key: "_id", Value: gardenId},
	}
	gardenRes, gardenErr := collection.DeleteOne(context.TODO(), query)

	if gardenErr != nil {
		log.Println(gardenErr)
	}

	_, rulesErr := rules_controllers.DeleteRulesByGardenId(gardenId)

	if rulesErr != nil {
		log.Println("Error deleting rules for ID", gardenId, rulesErr)
	}

	return gardenRes, gardenErr
}
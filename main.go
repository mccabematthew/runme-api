package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// --- Structs ---

type RunTable struct {
	DynamoDbClient *dynamodb.Client
	TableName      string
}

type Run struct {
	UserID          string  `json:"userId" dynamodbav:"userId"`
	RunID           string  `json:"runId" dynamodbav:"runId"`
	Date            string  `json:"date" dynamodbav:"date"`
	DistanceMiles   float64 `json:"distance_miles" dynamodbav:"distance_miles"`
	DurationSeconds int     `json:"duration_seconds" dynamodbav:"duration_seconds"`
	RunType         string  `json:"run_type" dynamodbav:"run_type"`
	Feel            int     `json:"feel" dynamodbav:"feel"`
}

// --- DynamoDB Operations ---

func (entry RunTable) AddRun(ctx context.Context, run Run) error {
	item, err := attributevalue.MarshalMap(run)
	if err != nil {
		panic(err)
	}
	_, err = entry.DynamoDbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(entry.TableName),
		Item:      item,
	})
	if err != nil {
		log.Printf("Couldn't add item to table. Here's why: %v\n", err)
	}
	return err
}

func (entry RunTable) GetRun(ctx context.Context, userID string, runID string) (Run, error) {
	key := map[string]types.AttributeValue{
		"userId": &types.AttributeValueMemberS{Value: userID},
		"runId":  &types.AttributeValueMemberS{Value: runID},
	}
	response, err := entry.DynamoDbClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(entry.TableName),
		Key:       key,
	})
	if err != nil {
		log.Printf("Couldn't find item in table. Here's why: %v\n", err)
		return Run{}, err
	}
	var run Run
	err = attributevalue.UnmarshalMap(response.Item, &run)
	if err != nil {
		panic(err)
	}
	return run, nil
}

func (entry RunTable) ListRuns(ctx context.Context, userID string) ([]Run, error) {
	response, err := entry.DynamoDbClient.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(entry.TableName),
		KeyConditionExpression: aws.String("userId = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		log.Printf("Couldn't list runs. Here's why: %v\n", err)
		return nil, err
	}
	var runs []Run
	err = attributevalue.UnmarshalListOfMaps(response.Items, &runs)
	if err != nil {
		log.Printf("Couldn't unmarshal runs. Here's why: %v\n", err)
		return nil, err
	}
	return runs, nil
}

// --- Lambda Handler ---

var table RunTable

func init() {
	ctx := context.Background()
	sdkConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		log.Fatalf("Couldn't load AWS config: %v", err)
	}
	table = RunTable{
		DynamoDbClient: dynamodb.NewFromConfig(sdkConfig),
		TableName:      os.Getenv("TABLE_NAME"),
	}
}

func handler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	if request.RequestContext.HTTP.Method == "OPTIONS" {
		return events.APIGatewayV2HTTPResponse{StatusCode: 200, Headers: headers}, nil
	}

	switch request.RequestContext.HTTP.Method {
	case "POST":
		return handleAddRun(ctx, request, headers)
	case "GET":
		if runID, ok := request.QueryStringParameters["runId"]; ok {
			return handleGetRun(ctx, request, runID, headers)
		}
		return handleListRuns(ctx, request, headers)
	default:
		return events.APIGatewayV2HTTPResponse{StatusCode: 405, Headers: headers}, nil
	}
}

func handleAddRun(ctx context.Context, request events.APIGatewayV2HTTPRequest, headers map[string]string) (events.APIGatewayV2HTTPResponse, error) {
	var run Run
	if err := json.Unmarshal([]byte(request.Body), &run); err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 400,
			Headers:    headers,
			Body:       `{"error": "invalid request body"}`,
		}, nil
	}

	// TODO: replace with real userId from Cognito JWT claims
	run.UserID = "user_matt"

	if err := table.AddRun(ctx, run); err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Headers:    headers,
			Body:       `{"error": "failed to save run"}`,
		}, nil
	}

	body, _ := json.Marshal(run)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: 201,
		Headers:    headers,
		Body:       string(body),
	}, nil
}

func handleGetRun(ctx context.Context, request events.APIGatewayV2HTTPRequest, runID string, headers map[string]string) (events.APIGatewayV2HTTPResponse, error) {
	// TODO: replace with real userId from Cognito JWT claims
	userID := "user_matt"

	run, err := table.GetRun(ctx, userID, runID)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 404,
			Headers:    headers,
			Body:       `{"error": "run not found"}`,
		}, nil
	}

	body, _ := json.Marshal(run)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Headers:    headers,
		Body:       string(body),
	}, nil
}

func handleListRuns(ctx context.Context, request events.APIGatewayV2HTTPRequest, headers map[string]string) (events.APIGatewayV2HTTPResponse, error) {
	// TODO: replace with real userId from Cognito JWT claims
	userID := "user_matt"

	runs, err := table.ListRuns(ctx, userID)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{
			StatusCode: 500,
			Headers:    headers,
			Body:       `{"error": "failed to list runs"}`,
		}, nil
	}

	body, _ := json.Marshal(runs)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: 200,
		Headers:    headers,
		Body:       string(body),
	}, nil
}

func main() {
	lambda.Start(handler)
}

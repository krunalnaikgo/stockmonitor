package utils

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/ses"
)

type SNSDetails struct {
	AwsRegion string
	ToEmail   string
	FromEmail string
	Subject   string
	CharSet   string
	TextBody  string
}

type QeuryStock struct {
	StockName string `json:"stockName" dynamodbav:"stockName"`
	OpenVal   string `json:"OpenVal" dynamodbav:"OpenVal"`
	CloseVal  string `json:"CloseVal" dynamodbav:"CloseVal"`
	HighPrice string `json:"HighPrice" dynamodbav:"HighPrice"`
	HighPriceDate string `json:"HighPriceDate" dynamodbav:"HighPriceDate"`
}

func GetCurrentTime() string {
	currentTime := time.Now()
	date := currentTime.Format("2006-01-02")
	fmt.Println(date)
	return date
}

func SendEmail(toEmail string, fromEmail string, emailPassword string, message string) {
	subject := "Stock Notification for Today\n\n"

	body := "From: " + fromEmail + "\n" +
		"To: " + toEmail + "\n" +
		"Subject:" + subject + message
	auth := smtp.PlainAuth("", fromEmail, emailPassword, "smtp.gmail.com")
	err := smtp.SendMail(
		"smtp.gmail.com:587",
		auth,
		fromEmail,
		[]string{toEmail},
		[]byte(body),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func GetEnvValue(envValue string) string {
	var valueName string
	value, con := os.LookupEnv(envValue)
	if con {
		log.Println("Found Environment Variables:", envValue)
		valueName = value
	} else {
		log.Fatalln("Value is not found", envValue)
	}
	return valueName
}

func (snsObj *SNSDetails) SendSNSEmail() {
	// Create a new session in the us-west-2 region.
	// Replace us-west-2 with the AWS Region you're using for Amazon SES.
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(snsObj.AwsRegion)},
	)

	// Create an SES session.
	svc := ses.New(sess)

	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(snsObj.ToEmail),
			},
		}, Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Charset: aws.String(snsObj.CharSet),
					Data:    aws.String(snsObj.TextBody),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(snsObj.CharSet),
				Data:    aws.String(snsObj.Subject),
			},
		},
		Source: aws.String(snsObj.FromEmail),
		// Uncomment to use a configuration set
		//ConfigurationSetName: aws.String(ConfigurationSet),
	}

	// Attempt to send the email.
	result, err := svc.SendEmail(input)

	// Display error messages if they occur.
	// Display error messages if they occur.
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ses.ErrCodeMessageRejected:
				fmt.Println(ses.ErrCodeMessageRejected, aerr.Error())
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				fmt.Println(ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				fmt.Println(ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}

		return
	}

	fmt.Println("Email Sent to address: " + snsObj.ToEmail)
	fmt.Println(result)
}

func getS3Buckets(awsRegion string) []string {
	var foundBuckets []string
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion)},
	)

	if err != nil {
		log.Fatal("Got Error getS3Buckets: ", err)
	}

	// Create S3 service client
	svc := s3.New(sess)
	result, err := svc.ListBuckets(nil)
	if err != nil {
		log.Fatal("Unable to list buckets, %v", err)
	}

	for _, b := range result.Buckets {
		foundBuckets = append(foundBuckets, aws.StringValue(b.Name))
		fmt.Printf("* %s created on %s\n",
			aws.StringValue(b.Name), aws.TimeValue(b.CreationDate))
	}
	return foundBuckets
}

func getDynamodbTable(awsRegion string, tableName string) bool {
	var foundTable bool
	newsess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion)},
	)

	if err != nil {
		log.Fatal("Got Error getS3Buckets: ", err)
	}

	svc := dynamodb.New(newsess)
	req := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}
	_, err1 := svc.DescribeTable(req)
	if err1 != nil {
		foundTable = false
	} else {
		foundTable = true
	}
	return foundTable
}

func CreateDynamodbTable(awsRegion string, tableName string) {
	tableFound := getDynamodbTable(awsRegion, tableName)
	if !tableFound {
		newsess, err := session.NewSession(&aws.Config{
			Region: aws.String(awsRegion)},
		)

		if err != nil {
			log.Fatal("Got Error getS3Buckets: ", err)
		}

		svc := dynamodb.New(newsess)
		input := &dynamodb.CreateTableInput{
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("stockName"),
					AttributeType: aws.String("S"),
				},
			},
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("stockName"),
					KeyType:       aws.String("HASH"),
				},
			},
			ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(10),
				WriteCapacityUnits: aws.Int64(10),
			},
			TableName: aws.String(tableName),
		}

		result, err := svc.CreateTable(input)
		if err != nil {
			log.Fatalln("Found Error", err)
		}

		log.Println("Created Table", result)

	} else {
		log.Println("Dynamodb Table is already created", tableName)
	}
}

func UpdateTable(awsRegion string, tableName string, openVal string, closeVal string,
	highPrice string, stockName string, HighPriceDate string) {
	newsess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion)},
	)

	if err != nil {
		log.Fatal("Got Error getS3Buckets: ", err)
	}

	svc := dynamodb.New(newsess)
	params := &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]*dynamodb.AttributeValue{
			"stockName": {
				S: aws.String(stockName),
			},
		},
		UpdateExpression: aws.String("set OpenVal = :c, CloseVal = :p, HighPrice= :r, HighPriceDate= :q"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":c": {S: aws.String(openVal)},
			":p": {S: aws.String(closeVal)},
			":r": {S: aws.String(highPrice)},
			":q": {S: aws.String(HighPriceDate)},
		},
		ReturnValues: aws.String(dynamodb.ReturnValueAllNew),
	}
	resp, err := svc.UpdateItem(params)
	if err != nil {
		log.Fatalln("Got Error", err)
	}
	log.Println(resp)
}

func QueryTable(awsRegion string, tableName string, stockName string) (float64, float64, float64, string, string) {
	newsess, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion)},
	)

	if err != nil {
		log.Fatal("Got Error getS3Buckets: ", err)
	}

	svc := dynamodb.New(newsess)
	queryInput := &dynamodb.QueryInput{
		TableName: aws.String(tableName),
		KeyConditions: map[string]*dynamodb.Condition{
			"stockName": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(stockName),
					},
				},
			},
		},
	}

	resp, err := svc.Query(queryInput)
	if err != nil {
		log.Fatalln("Found Error", err)
	}
	var stockQuery []QeuryStock
	err2 := dynamodbattribute.UnmarshalListOfMaps(resp.Items, &stockQuery)
	if err2 != nil {
		log.Fatalln("Found Error", err2)
	}

	var foundOpen float64
	var foundClose float64
	var foundHighPrice float64
	var foundStock string
	var foundHighPriceDate string

	for _, m := range stockQuery {
		foundOpen, _ = strconv.ParseFloat(strings.TrimSpace(m.OpenVal), 64)
		foundClose, _ = strconv.ParseFloat(strings.TrimSpace(m.CloseVal), 64)
		foundHighPrice, _ = strconv.ParseFloat(strings.TrimSpace(m.HighPrice), 64)
		foundStock = m.StockName
		foundHighPriceDate = m.HighPriceDate
	}

	return foundOpen, foundClose, foundHighPrice, foundStock, foundHighPriceDate
}

func CheckIncreaseValues(prevValue float64, currValue float64) bool {
	var isIncrease bool
	if currValue > prevValue {
		isIncrease = true
	} else {
		isIncrease = false
	}
	return isIncrease
}

func CalculateProfitOrLoss(boughtValue float64, currValue float64, stockSize float64) float64 {
	calcDiff := currValue - boughtValue
	calcGain := calcDiff * stockSize
	return calcGain
}

func createBucket(awsRegion string, cBucketName string) {
	listBuckets := getS3Buckets(awsRegion)
	var foundBucket bool
	for _, bucketName := range listBuckets {
		if bucketName == cBucketName {
			foundBucket = true
		}
	}

	if foundBucket {
		log.Println("Bucket Already Exists", cBucketName)
	} else {
		sess, err := session.NewSession(&aws.Config{
			Region: aws.String(awsRegion)},
		)

		if err != nil {
			log.Fatal("Got Error createBucket: ", err)
		}

		svc := s3.New(sess)

		_, err = svc.CreateBucket(&s3.CreateBucketInput{
			Bucket: aws.String(cBucketName),
		})
		if err != nil {
			log.Fatalf("Unable to create bucket %q, %v", cBucketName, err)
		}

		// Wait until bucket is created before finishing
		log.Printf("Waiting for bucket %q to be created...\n", cBucketName)

		err = svc.WaitUntilBucketExists(&s3.HeadBucketInput{
			Bucket: aws.String(cBucketName),
		})
		if err != nil {
			log.Fatalf("Error occurred while waiting for bucket to be created, %v", cBucketName)
		}

		log.Printf("Bucket %q successfully created\n", cBucketName)
	}
}

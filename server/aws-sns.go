package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type Notification struct {
	UserEmail string
	SpaceID   string
	BookingID string
	UserID    string
	StartTime time.Time
	EndTime   time.Time
	Status    string
	Location  string
	Space     string
}

type AWSSns struct {
	Client   *sns.Client
	TopicARN string
}

var _snsClientOnce sync.Once
var _snsClientInstance *AWSSns

func GetSnsClient() *AWSSns {
	_snsClientOnce.Do(func() {
		_snsClientInstance = &AWSSns{}
		_snsClientInstance.Open()
	})
	return _snsClientInstance
}

func (snsAWS *AWSSns) Open() {
	log.Println("Connecting to AWS...")
	config := GetConfig()
	staticProvider := credentials.NewStaticCredentialsProvider(
		config.AWSAccessKey,
		config.AWSSecretAccessKey,
		config.AWSSessionToken,
	)
	cfg, err := awsConfig.LoadDefaultConfig(
		context.Background(),
		awsConfig.WithCredentialsProvider(staticProvider),
		awsConfig.WithRegion(config.AWSRegion),
		awsConfig.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(retry.NewStandard(), 5)
		}),
	)
	if err != nil {
		log.Println("Couldn't load default configuration. Have you set up your AWS account?")
	}
	log.Println("AWS Config loaded...")
	snsAWS.Client = sns.NewFromConfig(cfg)
	topicARN, err := snsAWS.GetTopicARN(config.AWSSNSTopic)
	if err != nil {
		log.Printf("Could not get Topic ARN for '%v'", config.AWSSNSTopic)
	}

	snsAWS.TopicARN = topicARN
}

func (snsAWS *AWSSns) ListTopics() ([]types.Topic, error) {
	var topics []types.Topic
	paginator := sns.NewListTopicsPaginator(snsAWS.Client, &sns.ListTopicsInput{})
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			log.Printf("Couldn't get topics. Here's why: %v\n", err)
			return nil, err
		} else {
			topics = append(topics, output.Topics...)
		}
	}
	log.Printf("Length of Topics: %v", len(topics))
	return topics, nil
}

func (snsAWS *AWSSns) GetTopicARN(topic string) (string, error) {
	resp, err := snsAWS.Client.CreateTopic(context.Background(), &sns.CreateTopicInput{Name: &topic})
	if err != nil {
		log.Printf("Couldn't get topics")
		return "", err
	}
	return *resp.TopicArn, nil
}

func (snsAWS *AWSSns) PushBookingEvent(booking *Booking, userEmail string, space string, location, status string) (bool, error) {

	notification := Notification{}
	notification.UserEmail = userEmail
	notification.UserID = booking.UserID
	notification.SpaceID = booking.SpaceID
	notification.BookingID = booking.ID
	notification.StartTime = booking.Enter
	notification.EndTime = booking.Leave
	notification.Space = space
	notification.Status = status
	notification.Location = location

	bookingStr, err := json.Marshal(&notification)
	if err != nil {
		return false, err
	}
	log.Printf("Pushing Create event to SNS topic: %v", snsAWS.TopicARN)
	req, err := snsAWS.Client.Publish(context.Background(), &sns.PublishInput{TopicArn: &snsAWS.TopicARN, Message: aws.String(string(bookingStr))})
	if err != nil {
		log.Printf("Could not push to SNS, %v", err)
		return false, err
	}
	log.Printf("Message response from Push even %v", req.MessageId)

	return true, nil
}

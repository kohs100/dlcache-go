package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

//{
//	"params": {
//	  "path": {
//		"rjcode": "VJ014646"
//	  }
//	}
//}

type CodeObj struct {
	Code string `json:"rjcode"`
}

type Paths struct {
	Path CodeObj `json:"path"`
}

type GETEvent struct {
	Params Paths `json:"params"`
}

type WorkResponse struct {
	Product_id string `json:"productId"`
	Title      string `json:"title"`
	Maker      string `json:"maker"`
	Category   string `json:"category"`
	Date       string `json:"releaseDate"`
	Img_url    string `json:"imgURI"`
	Req_url    string `json:"reqURI"`
}

func IsRJCode(str string) bool {
	if len(str) != 8 {
		return false
	}

	if str[:2] != "RJ" && str[:2] != "VJ" && str[:2] != "BJ" {
		return false
	}

	_, err := strconv.Atoi(str[2:])
	return err == nil
}

var sess *session.Session
var svcS3 *s3.S3
var svcDynamoDB *dynamodb.DynamoDB

func init() {
	sess = session.Must(session.NewSession())
	svcS3 = s3.New(sess)
	svcDynamoDB = dynamodb.New(sess)
}

func GetImgURI(richcode string) (bool, string, error) {
	item := WorkResponse{}

	res, err := svcDynamoDB.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("dlsite-metadata"),
		Key: map[string]*dynamodb.AttributeValue{
			"productId": {
				S: aws.String(richcode),
			},
		},
	})
	if err != nil {
		return false, "", err
	}

	if len(res.Item) == 0 {
		return false, "", nil
	}

	err = dynamodbattribute.UnmarshalMap(res.Item, &item)
	if err != nil {
		return false, "", err
	}

	return true, item.Img_url, nil
}

func HeadS3(fname string) bool {
	_, err := svcS3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String("dlsite-thumbs"),
		Key:    aws.String(fname),
	})
	if err != nil {
		return false
	}
	return true
}

func GetS3(fname string) (io.Reader, bool) {
	res, err := svcS3.GetObject(&s3.GetObjectInput{
		Bucket: aws.String("dlsite-thumbs"),
		Key:    aws.String(fname),
	})
	if err != nil {
		return nil, false
	}
	return res.Body, true
}

func PostS3(fname string, f io.Reader) error {
	uploader := s3manager.NewUploader(sess)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("dlsite-thumbs"),
		Key:    aws.String(fname),
		Body:   f,
	})
	return err
}

/*
func GetS3URL(fname string) (string, error) {
	req, _ := svcS3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String("dlsite-thumbs"),
		Key:    aws.String(fname),
	})
	urlStr, err := req.Presign(15 * time.Minute)
	if err != nil {
		return "", err
	}
	return urlStr, nil
}
*/

func getFileName(uri string) string {
	tokens := strings.Split(uri, "/")
	return tokens[len(tokens)-1]
}

/*
func buildRedirection(uri string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: 302,
		Headers: map[string]string{
			"Location": uri,
		},
	}
}
*/

func buildNotFoundError(err string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: 404,
		Body:       "NotFoundError: " + err,
	}
}

func buildInternalError(err string) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		StatusCode: 500,
		Body:       "InternalError: " + err,
	}
}

func buildBlobResponse(bytes []byte) events.APIGatewayProxyResponse {
	sEnc := base64.StdEncoding.EncodeToString(bytes)
	return events.APIGatewayProxyResponse{
		StatusCode:      200,
		Body:            sEnc,
		IsBase64Encoded: true,
		Headers: map[string]string{
			"Access-Control-Allow-Origin": "*",
			"Content-Type":                "image/jpeg",
		},
	}
}

func parseRequest(request events.APIGatewayProxyRequest) string {
	tokens := strings.Split(request.Path, "/")
	return tokens[len(tokens)-2]
}

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	code := parseRequest(request)

	if !IsRJCode(code) {
		return buildNotFoundError(code), nil
	}

	ondb, img_uri, err := GetImgURI(code)
	if err != nil {
		return buildInternalError("GetDB Failed"), nil
	}
	if !ondb {
		return buildNotFoundError("Work Metadata Not Found"), nil
	}

	img_name := getFileName(img_uri)

	img_body, ons3 := GetS3(img_name)

	if ons3 {
		img_bytes, _ := io.ReadAll(img_body)
		return buildBlobResponse(img_bytes), nil
	}

	res, err := http.Get(img_uri)

	if err != nil {
		return buildInternalError("GET img_uri Failed"), nil
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return buildInternalError("GET img_uri Status: " + res.Status), nil
	}

	img_bytes, _ := io.ReadAll(res.Body)

	err = PostS3(img_name, bytes.NewReader(img_bytes))
	if err != nil {
		return buildInternalError("POST img_name failed"), nil
	}

	return buildBlobResponse(img_bytes), nil
}

func main() {
	lambda.Start(HandleRequest)
}

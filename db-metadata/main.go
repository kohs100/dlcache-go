package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
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

var svc *dynamodb.DynamoDB

func init() {
	sess := session.Must(session.NewSession())
	svc = dynamodb.New(sess)
}

func GetDB(richcode string) (bool, WorkResponse, error) {
	item := WorkResponse{}

	res, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("dlsite-metadata"),
		Key: map[string]*dynamodb.AttributeValue{
			"productId": {
				S: aws.String(richcode),
			},
		},
	})
	if err != nil {
		return false, item, err
	}

	if len(res.Item) == 0 {
		return false, item, nil
	}

	err = dynamodbattribute.UnmarshalMap(res.Item, &item)
	if err != nil {
		return false, item, err
	}
	return true, item, nil
}

func PushDB(work WorkResponse) error {
	av, err := dynamodbattribute.MarshalMap(work)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("dlsite-metadata"),
	}

	_, err = svc.PutItem(input)
	return err
}

func getDateAsISO(doc *goquery.Document) (string, bool) {
	val := ""
	doc.Find("#work_outline a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		value, isExist := s.Attr("href")
		if isExist {
			tokens := strings.Split(value, "/")
			if len(tokens) < 12 {
				return true
			}
			if tokens[6] == "year" {
				val = tokens[7] + "-" + tokens[9] + "-" + tokens[11]
				return false
			}
		}
		return true
	})
	if len(val) == 0 {
		return "", false
	} else {
		return val, true
	}
}

func HandleRequest(ctx context.Context, event GETEvent) (WorkResponse, error) {
	code := event.Params.Path.Code
	item := WorkResponse{}

	if !IsRJCode(code) {
		return item, errors.New("NotFoundError: Invalid RJCode")
	}

	ondb, resDB, err := GetDB(code)
	if err != nil {
		return item, errors.New("InternalError: GetDB Failed")
	}

	if ondb {
		return resDB, nil
	}

	url := fmt.Sprintf("https://www.dlsite.com/soft/work/=/product_id/%s.html", code)
	res, err := http.Get(url)

	if err != nil {
		return item, errors.New("InternalError: GET Failed")
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return item, errors.New("NotFoundError: Work Not Found")
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)

	dateiso, found := getDateAsISO(doc)

	if !found {
		return item, errors.New("InternalError: Date Parsing Failed")
	}

	title := doc.FindMatcher(goquery.Single("h1#work_name > a")).Text()
	imgURL, _ := doc.FindMatcher(goquery.Single("li.slider_item.active > img")).Attr("src")
	maker := doc.FindMatcher(goquery.Single("span.maker_name> a")).Text()
	finalURL := res.Request.URL.String()
	category := strings.Split(finalURL, "/")[3]

	resp := &WorkResponse{code, title, maker, category, dateiso, "https:" + imgURL, finalURL}

	err = PushDB(*resp)
	if err != nil {
		return item, errors.New("InternalError: PushDB Failed")
	}

	return *resp, nil
}

func main() {
	lambda.Start(HandleRequest)
}

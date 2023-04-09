/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE', which is part of this source code package.
 */

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/ipthomas/tukcnst"
	"github.com/ipthomas/tukdbint"
	"github.com/ipthomas/tukxdw"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var initSrvcs = false

func main() {
	lambda.Start(Handle_Request)
}
func Handle_Request(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	log.SetFlags(log.Lshortfile)

	var err error
	if !initSrvcs {
		dbconn := tukdbint.TukDBConnection{DBUser: os.Getenv(tukcnst.ENV_DB_USER), DBPassword: os.Getenv(tukcnst.ENV_DB_PASSWORD), DBHost: os.Getenv(tukcnst.ENV_DB_HOST), DBPort: os.Getenv(tukcnst.ENV_DB_PORT), DBName: os.Getenv(tukcnst.ENV_DB_NAME)}
		if err = tukdbint.NewDBEvent(&dbconn); err != nil {
			return queryResponse(http.StatusInternalServerError, err.Error(), tukcnst.TEXT_PLAIN)
		}
		initSrvcs = true
	}

	log.Printf("Processing API Gateway %s Request Path %s", req.HTTPMethod, req.Path)
	var task string
	trans := tukxdw.Transaction{}
	for key, value := range req.QueryStringParameters {
		log.Printf("    %s: %s\n", key, value)
		switch key {
		case tukcnst.TASK:
			task = value
		case tukcnst.TUK_EVENT_QUERY_PARAM_PATHWAY:
			trans.Pathway = value
		case tukcnst.TUK_EVENT_QUERY_PARAM_NHS:
			trans.NHS_ID = value
		}
	}
	if req.HTTPMethod == http.MethodPost {
		switch task {
		case "def":
			trans.Actor = tukcnst.XDW_ADMIN_REGISTER_DEFINITION
		case "meta":
			trans.Actor = tukcnst.XDW_ADMIN_REGISTER_XDS_META
		}
		trans.DSUB_BrokerURL = os.Getenv(tukcnst.ENV_DSUB_BROKER_URL)
		trans.DSUB_ConsumerURL = os.Getenv(tukcnst.ENV_DB_USER)
		trans.Request = []byte(req.Body)
		tukxdw.Execute(&trans)
		if task == "meta" {
			return queryResponse(http.StatusOK, "", tukcnst.TEXT_PLAIN)
		}
	}
	trans.Actor = tukcnst.XDW_ACTOR_CONTENT_CREATOR
	if err = tukxdw.Execute(&trans); err == nil {
		var wfs []byte
		if wfs, err = json.MarshalIndent(trans.XDWDocument, "", "  "); err == nil {
			return queryResponse(http.StatusOK, string(wfs), tukcnst.APPLICATION_JSON)
		}
	}
	return queryResponse(http.StatusInternalServerError, err.Error(), tukcnst.TEXT_PLAIN)
}
func setAwsResponseHeaders(contentType string) map[string]string {
	awsHeaders := make(map[string]string)
	awsHeaders["Server"] = "TUK_XDW_Consumer_Proxy"
	awsHeaders["Access-Control-Allow-Origin"] = "*"
	awsHeaders["Access-Control-Allow-Headers"] = "accept, Content-Type"
	awsHeaders["Access-Control-Allow-Methods"] = "GET, POST, OPTIONS"
	awsHeaders[tukcnst.CONTENT_TYPE] = contentType
	return awsHeaders
}
func queryResponse(statusCode int, body string, contentType string) (*events.APIGatewayProxyResponse, error) {
	log.Println(body)
	return &events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    setAwsResponseHeaders(contentType),
		Body:       body,
	}, nil
}

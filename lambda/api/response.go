package api

import (
	"encoding/base64"
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"net/http"
)

type Response struct {
	Message string `json:"message"`
}

func response(obj interface{}, status int) events.APIGatewayV2HTTPResponse {
	b, _ := json.Marshal(obj)
	return events.APIGatewayV2HTTPResponse{StatusCode: status, Body: string(b)}
}

func OK() events.APIGatewayV2HTTPResponse {
	return response(&Response{Message: "OK"}, 200)
}

func OKBytes(b []byte) events.APIGatewayV2HTTPResponse {
	msg := base64.StdEncoding.EncodeToString(b)
	return events.APIGatewayV2HTTPResponse{
		StatusCode:      http.StatusOK,
		Body:            msg,
		IsBase64Encoded: true,
	}
}

func InternalError() events.APIGatewayV2HTTPResponse {
	return response(&Response{Message: "internal error"}, http.StatusInternalServerError)
}

func NotFound() events.APIGatewayV2HTTPResponse {
	return response(&Response{Message: "not found"}, http.StatusNotFound)
}

func Unauthorized() events.APIGatewayV2HTTPResponse {
	return response(&Response{Message: "unauthorized"}, http.StatusUnauthorized)
}

func MethodNotAllowed() events.APIGatewayV2HTTPResponse {
	return response(&Response{Message: "method not allowed"}, http.StatusMethodNotAllowed)
}

func BadRequest(msg string) events.APIGatewayV2HTTPResponse {
	return response(&Response{Message: msg}, http.StatusBadRequest)
}

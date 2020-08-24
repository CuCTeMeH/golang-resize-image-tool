package main

import (
	"github.com/CuCTeMeH/golang-resize-image-tool/handlers"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	lambda.Start(new(handlers.GatewayHandler).ServeHTTP)
}

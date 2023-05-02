.PHONY: build clean deploy

build:
	cd lambda-bot-db && env GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/lambda-bot-db handler.go && cd ..

test:
	cd lambda-bot-db && go test ./... && cd ..

clean:
	rm -rf ./bin ./vendor Gopkg.lock

deploy: clean build
	sls deploy --stage prod --verbose

deploy-research: clean build
	sls deploy --aws-profile forta-research --stage research --verbose

mocks:
	mockgen -source lambda-bot-db/store/s3.go -destination lambda-bot-db/store/mocks/mock_s3.go
	mockgen -source lambda-bot-db/store/dynamodb.go -destination lambda-bot-db/store/mocks/mock_dynamodb.go

.PHONY: build clean deploy

build:
	cd lambda && env GOOS=linux CGO_ENABLED=0 go build -ldflags="-s -w" -o ../bin/lambda handler.go && cd ..

test:
	cd lambda && go test ./... && cd ..

clean:
	rm -rf ./bin ./vendor Gopkg.lock

deploy: clean build
	sls deploy --stage prod --verbose

deploy-research: clean build
	sls deploy --aws-profile forta-research --stage research --verbose

mocks:
	mockgen -source lambda/store/s3.go -destination lambda/store/mocks/mock_s3.go
	mockgen -source lambda/store/dynamodb.go -destination lambda/store/mocks/mock_dynamodb.go

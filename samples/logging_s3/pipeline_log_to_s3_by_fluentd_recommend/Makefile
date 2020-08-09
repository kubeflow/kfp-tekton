all: build push

build:
	docker build -t fenglixa/pipeline-logs-s3:latest .

push:
	docker push fenglixa/pipeline-logs-s3:latest

deploy:
	@echo 'Deleting existing deployment'
	@kubectl delete -f pipeline-logs-fluentd-s3.yaml || echo 'No deployment found, carrying on'
	@echo 'Creating new deployment'
	@kubectl create -f pipeline-logs-fluentd-s3.yaml

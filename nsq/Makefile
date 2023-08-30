infra/ansible/build:
	mkdir ./infra/ansible/build

build: infra/ansible/build/nsqload infra/ansible/build/nsqsub
infra/ansible/build/nsqload: infra/ansible/build cmd/nsqload
	go build -o infra/ansible/build ./cmd/nsqload

infra/ansible/build/nsqsub: infra/ansible/build cmd/nsqsub
	go build -o infra/ansible/build ./cmd/nsqsub

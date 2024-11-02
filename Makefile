
IMAGE=kube-burst
RELEASE=v1.0.0

bin:
	 CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build -o kube-burst ./cmd/...


image: bin
	docker build -t ${IMAGE}:${RELEASE} .
	docker push ${IMAGE}:${RELEASE} 


all: image copy

image:
	docker build -t huproxy .

copy:
	docker rm huproxy 2>/dev/null || true
	docker create --name huproxy huproxy
	docker cp huproxy:/opt/huproxy ./
	docker cp huproxy:/opt/huproxyclient_linux_amd64 ./
	docker cp huproxy:/opt/huproxyclient_darwin_amd64 ./
	docker cp huproxy:/opt/huproxyclient_win64.exe ./
	docker rm huproxy

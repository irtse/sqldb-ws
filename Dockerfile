FROM golang as builder
 
WORKDIR /app

COPY . . 
 
ENV CGO_ENABLED=1

RUN rm -rf files

RUN go build -buildmode=plugin -o plugins/cegid/plugin.so plugins/cegid/plugin.go
RUN go build -buildmode=plugin -o plugins/autoload_cegid/plugin.so plugins/autoload_cegid/plugin.go

RUN sed -i 's/http:\/\/127.0.0.1:8080\/swagger\/swagger.json/swagger.json/g' swagger/index.html

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" .

EXPOSE 8080
 
ENTRYPOINT ["./sqldb-ws"]
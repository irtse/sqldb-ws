Generic database webservice layer
=================================
before doing go mod tidy.

To build :

    bee generate routers
    bee run -gendoc=true -downdoc=true

RUN GO PROJECT 
`go run main.go`
 
# DEV DOCKER LAUNCH

Prerequisite : Get docker on your local machine. 
Launch tools before  : 
    `docker compose -f docker-compose.tools.yml`

then api :
    `docker build -t irtse/sqldb-ws`
    `docker compose -f docker-compose.yml`

Super Admin SQLDB-WS Default
username : root
password : admin

{
  "login": "root",
  "password": "admin"
}

Super Admin DB Default
    Type : PostgresSQL
    Host: sqldb-ws-pg
    Database: test
    User : test
    Password : test

Grafana wait for first conn.

# PROD DOCKER LAUNCH

Prerequisite : Get docker on your local machine. 
Launch api PROD :
    `docker compose -f docker-compose.prod.yml`

Super Admin SQLDB-WS Default
username : root
password : imnotthepwd

{
  "login": "root",  (can be change at first start : SUPERADMIN_NAME)
  "password": "imnotthepwd"  (can be change at first start : SUPERADMIN_PASSWORD)
}

Super Admin DB Default
    Type : PostgresSQL
    Host: sqldb-ws-pg
    Database: opps (can be change at first start : DBNAME)
    User : opps (can be change at first start : DBUSER)
    Password : imnotthepwd (can be change at first start : DBPWD)

Grafana wait for first conn.

bee generate routers
bee run -gendoc=true -downdoc=true
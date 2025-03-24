FROM ubuntu:latest

WORKDIR /app

ADD helm-api .

COPY source/helm/mariadb ./source/helm/mariadb

EXPOSE 8080

CMD  /app/helm-api
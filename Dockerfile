# syntax=docker/dockerfile:1

FROM golang:1.17-bullseye AS BUILD
WORKDIR /app
COPY . .

RUN go mod download
RUN go build -o /thunder_balancer

FROM gcr.io/distroless/base-debian11
WORKDIR /
COPY --from=build /thunder_balancer /thunder_balancer
EXPOSE 3000
CMD ["/thunder_balancer"]

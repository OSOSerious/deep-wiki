FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o product-catalog .

FROM gcr.io/distroless/static
COPY --from=build /src/product-catalog /product-catalog
EXPOSE 8080
ENTRYPOINT ["/product-catalog"]
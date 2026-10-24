FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /profile-cards .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /profile-cards /profile-cards
ENV PORT=8000
EXPOSE 8000
USER nonroot
ENTRYPOINT ["/profile-cards"]

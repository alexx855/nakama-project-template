FROM alexx855/nakama-pluginbuilder:5.0.2 AS builder

ENV GO111MODULE on
ENV CGO_ENABLED 1
ENV GOPRIVATE "github.com/alexx855/*"

WORKDIR /backend
COPY . .

# RUN go mod vendor
# docker run --rm -w "/builder" -v "${PWD}:/builder" alexx855/nakama-pluginbuilder:5.0.2 mod vendor
RUN go build --trimpath --mod=vendor --buildmode=plugin -o ./backend.so

FROM alexx855/nakama:5.0.9

COPY --from=builder /backend/backend.so /nakama/data/modules
COPY --from=builder /backend/service-account.json /nakama/data/modules
# COPY --from=builder /backend/*.lua /nakama/data/modules/
COPY --from=builder /backend/build/*.js /nakama/data/modules/build/
COPY --from=builder /backend/local.yml /nakama/data/

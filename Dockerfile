FROM heroiclabs/nakama-pluginbuilder:3.1.1 AS builder 
# docker build "$PWD" --build-arg commit="$(git rev-parse --short HEAD)" --build-arg version="$(git rev-parse --short HEAD)" -t heroiclabs/nakama-prerelease:"$(git rev-parse --short HEAD)"
# docker build . -t medievalgods/medievalgods-backend
ENV GO111MODULE on
ENV CGO_ENABLED 1
# ENV GOPRIVATE "github.com/medievalgods/*"

WORKDIR /backend
# ? organize in folders and dont copy all 
COPY . .

# RUN go mod vendor
# docker run --rm -w "/builder" -v "${PWD}:/builder" heroiclabs/nakama-pluginbuilder:3.1.1 mod vendor
RUN go build --trimpath --mod=vendor --buildmode=plugin -o ./backend.so

# ? Build js module
# FROM node
# WORKDIR /backend
# COPY . .
# RUN npm ci && npm run tsc

FROM heroiclabs/nakama:3.1.1
# Lua module
# COPY --from=builder /backend/*.lua /nakama/data/modules/

# Go module
COPY --from=builder /backend/backend.so /nakama/data/modules

# TS module
COPY --from=builder /backend/build/*.js /nakama/data/modules/build/

# Nakama config
COPY --from=builder /backend/local.yml /nakama/data/

# POSTGRES_DB
# POSTGRES_USER
# POSTGRES_PASSWORD
# CMD [ "executable" ]
# - /bin/sh
# - -ecx
# - |
# /nakama/nakama migrate up --database.address postgres:localdb@postgres:5432/nakama && exec /nakama/nakama --config /nakama/data/local.yml --database.address postgres:localdb@postgres:5432/nakama
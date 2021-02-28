FROM heroiclabs/nakama-pluginbuilder:3.1.1 AS builder 

ENV GO111MODULE on
ENV CGO_ENABLED 1
ENV GOPRIVATE "github.com/alexx855/*"

WORKDIR /backend
# ? organize in folders and dont copy all 
COPY . .

# RUN go mod vendor
# docker run --rm -w "/builder" -v "${PWD}:/builder" alexx855/nakama-pluginbuilder:5.0.2 mod vendor
RUN go build --trimpath --mod=vendor --buildmode=plugin -o ./backend.so

# FROM node
# WORKDIR /backend
# COPY . .
# RUN npm ci && npm run tsc

FROM heroiclabs/nakama:3.1.1
# TODO: move service account to enviroment
COPY --from=builder /backend/service-account.json /nakama/data/modules

COPY --from=builder /backend/backend.so /nakama/data/modules
# COPY --from=builder /backend/*.lua /nakama/data/modules/
# COPY --from=builder /backend/build/*.js /nakama/data/modules/build/
# COPY --from=node /backend/build/*.js /nakama/data/modules/build/
COPY --from=builder /backend/local.yml /nakama/data/

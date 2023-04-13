FROM heroiclabs/nakama-pluginbuilder:3.11.0 AS builder

ENV GO111MODULE on
ENV CGO_ENABLED 1

WORKDIR /backend
COPY . .

RUN go build --trimpath --mod=readonly --buildmode=plugin -o ./blackjack.so

FROM heroiclabs/nakama:3.11.0

COPY --from=builder /backend/bin/lobby.so /nakama/data/modules
COPY --from=builder /backend/blackjack.so /nakama/data/modules
COPY --from=builder /backend/local.yml /nakama/data/

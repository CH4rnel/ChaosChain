FROM node:22-alpine AS build

RUN apk add --no-cache gettext git
ARG PING_REF
ARG PUBLIC_URL
ARG CHAIN_ID
WORKDIR /src
RUN test "${#PING_REF}" -eq 40 \
 && git init \
 && git remote add origin https://github.com/ping-pub/explorer.git \
 && git fetch --depth 1 origin "${PING_REF}" \
 && test "$(git rev-parse FETCH_HEAD)" = "${PING_REF}" \
 && git checkout --detach FETCH_HEAD

COPY chains/chaoschain.json.template /tmp/chaoschain.json.template
RUN test -n "${PUBLIC_URL}" \
 && test -n "${CHAIN_ID}" \
 && mkdir -p chains/mainnet chains/testnet \
 && envsubst '${PUBLIC_URL} ${CHAIN_ID}' </tmp/chaoschain.json.template >chains/mainnet/chaoschain.json \
 && cp chains/mainnet/chaoschain.json chains/testnet/chaoschain.json \
 && corepack enable \
 && yarn install --frozen-lockfile --ignore-engines \
 && yarn build \
 && rm -rf .git node_modules

FROM nginx:stable-alpine
COPY --from=build /src/dist /usr/share/nginx/html
COPY nginx/explorer.conf.template /etc/nginx/templates/explorer.conf.template

FROM alpine:3.17.0

LABEL org.opencontainers.image.source=https://github.com/NordSecurity/nordvpn-linux

RUN apk add --no-cache curl jq bash git

CMD ["exec", "$@"]
FROM golang:1.24.2 AS build

COPY . /src
RUN cd /src && GOPROXY=https://goproxy.cn && CGO_ENABLED=0 make clean all

FROM scratch

COPY --from=build /src/bin/go /

EXPOSE 8067

ENTRYPOINT [ "/go" ]
CMD ["--data=/data"]

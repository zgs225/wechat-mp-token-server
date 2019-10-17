FROM golang:1.13
ADD . /app
RUN cd /app && go install && rm -rf /app
ENTRYPOINT ["wechat-mp-token-server"]

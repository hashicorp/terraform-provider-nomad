FROM golang:1.15.5

RUN apt-get update && \
    apt-get install -y \
        build-essential \
        httpie \
        jq
RUN mkdir /root/provider
COPY ./entrypoint.sh /entrypoint.sh
WORKDIR /root/provider
ENTRYPOINT ["/entrypoint.sh"]
CMD ["make", "testacc"]

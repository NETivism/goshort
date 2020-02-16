FROM netivism/docker-debian-base
LABEL maintainer="Jimmy Huang <jimmy@netivism.com.tw>"
LABEL description="This Dockerfile will install goshort"

EXPOSE 33512
COPY goshort /goshort
RUN chmod +x /goshort
CMD ["/goshort"]

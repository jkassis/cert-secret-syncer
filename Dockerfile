FROM scratch
EXPOSE 8080
ENTRYPOINT ["/cert-secret-syncer"]
COPY ./bin/ /
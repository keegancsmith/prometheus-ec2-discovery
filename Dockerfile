FROM golang:1.6-onbuild
ENTRYPOINT ["go-wrapper", "run"]

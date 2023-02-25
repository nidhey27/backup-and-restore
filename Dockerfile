FROM ubuntu

COPY ./backup-and-restore ./backup-and-restore

ENTRYPOINT [ "./backup-and-restore" ]
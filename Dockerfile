FROM fedora:25

ADD ./kube-burst /kube-burst

ENTRYPOINT ["/kube-burst"]


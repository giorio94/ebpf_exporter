# https://github.com/vanneback/ebpf_exporter_dockerfile/blob/782fd0bac75b5be49ef66c3b0036e8243e8b9be6/Dockerfile

FROM ubuntu:20.04 as builder

ENV DEBIAN_FRONTEND noninteractive

RUN apt update && apt install -y --no-install-recommends bison build-essential cmake flex git python3 python3-distutils \
    libedit-dev libllvm7 llvm-7-dev libclang-7-dev zlib1g-dev libelf-dev libfl-dev ca-certificates golang-1.16

WORKDIR /tmp

RUN git clone https://github.com/iovisor/bcc.git && \
    mkdir bcc/build; cd bcc/build && \
    cmake .. && \
    make && \
    make install

WORKDIR /tmp/builder

COPY go.mod ./go.mod
COPY go.sum ./go.sum
RUN  /usr/lib/go-1.16/bin/go mod download

COPY . ./
RUN  /usr/lib/go-1.16/bin/go build -ldflags="-s -w"  ./cmd/ebpf_exporter


FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive
RUN apt update && apt -y --no-install-recommends install libclang1-7 libelf1 && apt clean

COPY --from=builder /tmp/builder/ebpf_exporter /usr/local/bin/ebpf_exporter
COPY --from=builder /tmp/bcc/build/src/cc/libbcc.so.0.24.0 /usr/lib/x86_64-linux-gnu
COPY --from=builder /tmp/bcc/build/src/cc/libbcc_bpf.so.0.24.0 /usr/lib/x86_64-linux-gnu

RUN ldconfig -v -nN /usr/lib/x86_64-linux-gnu/ && \
    echo '#!/bin/bash' > /usr/local/bin/entrypoint.sh && \
    echo 'apt update && apt install -y -qq linux-headers-$(uname -r)' >> /usr/local/bin/entrypoint.sh && \
    echo 'exec /usr/local/bin/ebpf_exporter $@' >> /usr/local/bin/entrypoint.sh && \
    chmod 755 /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

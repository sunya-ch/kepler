FROM quay.io/sustainable_computing_io/kepler_builder:ubi-9-bcc-0.26-go1.18 as builder

ARG BIN_TIMESTAMP
ARG SOURCE_GIT_TAG

LABEL name=kepler-builder

ENV GOPATH=/opt/app-root GO111MODULE=on GOROOT=/usr/local/go 

ENV PATH=$GOPATH/bin:$GOROOT/bin:$PATH

WORKDIR $GOPATH/src/github.com/sustainable-computing-io/kepler

# TODO: remove after base image (Dockerfile.base) updated
RUN yum install -y clang llvm llvm-devel zlib-devel elfutils-libelf-devel make libbpf
RUN curl -LO https://github.com/libbpf/libbpf/archive/refs/tags/v0.5.0.tar.gz; mkdir -p /usr/local; tar -C /usr/local -xvzf v0.5.0.tar.gz; rm -f v0.5.0.tar.gz; cd /usr/local/libbpf-0.5.0/src && OBJDIR=/usr/lib/$(uname -m)-linux-gnu make install
RUN rm -rf /usr/local/go ; curl -LO https://go.dev/dl/go1.18.10.linux-amd64.tar.gz; mkdir -p /usr/local; tar -C /usr/local -xvzf go1.18.10.linux-amd64.tar.gz; rm -f go1.18.10.linux-amd64.tar.gz

COPY . .
RUN mkdir -p data

# Build kepler
RUN make build SOURCE_GIT_TAG=$SOURCE_GIT_TAG BIN_TIMESTAMP=$BIN_TIMESTAMP
RUN ls ./_output/bin
# RUN make test

# build image
FROM quay.io/sustainable_computing_io/kepler_base:ubi-9-bcc-0.26

COPY --from=builder /opt/app-root/src/github.com/sustainable-computing-io/kepler/_output/bin/kepler /usr/bin/kepler

RUN mkdir -p /var/lib/kepler/data
RUN mkdir -p /var/lib/kepler/bpfassets
COPY --from=builder /opt/app-root/src/github.com/sustainable-computing-io/kepler/data/normalized_cpu_arch.csv /var/lib/kepler/data/normalized_cpu_arch.csv
COPY --from=builder /opt/app-root/src/github.com/sustainable-computing-io/kepler/bpfassets/*.bpf.o /var/lib/kepler/bpfassets

ADD https://raw.githubusercontent.com/sustainable-computing-io/kepler-model-server/main/tests/test_models/DynComponentModelWeight/CgroupOnly/ScikitMixed/ScikitMixed.json /var/lib/kepler/data/ScikitMixed.json

ADD https://raw.githubusercontent.com/sustainable-computing-io/kepler-model-server/main/tests/test_models/AbsComponentModelWeight/Full/KerasCompWeightFullPipeline/KerasCompWeightFullPipeline.json /var/lib/kepler/data/KerasCompWeightFullPipeline.json

# pre install kernel sources
RUN mkdir -p /usr/share/kepler/kernel_sources

COPY --from=quay.io/sustainable_computing_io/kepler_kernel_source_images:ubi8 /usr/src/kernels /usr/share/kepler/kernel_sources
COPY --from=quay.io/sustainable_computing_io/kepler_kernel_source_images:ubi9 /usr/src/kernels /usr/share/kepler/kernel_sources

ENTRYPOINT ["/usr/bin/kepler"]

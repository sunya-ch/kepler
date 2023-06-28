ARCH=$(shell uname -m)

TARGET := kepler
TARGET_BPF := $(TARGET).bpf.o
BPF_SRC := assets/*.bpf.c

$(TARGET_BPF): $(BPF_SRC)
	clang \
		-I /usr/include/$(ARCH)-linux-gnu \
		-O2 -c -target bpf \
		-o assets/$(ARCH)_$@ $<
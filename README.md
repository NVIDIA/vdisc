VDisc
=====

![VDisc Logo](/doc/images/logo.svg)

VDisc is a tool for creating and mounting virtual CD-ROM images backed by object storage.

Building the VDisc CLI
---------------------

If you wish to work on VDisc itself or any of its libraries, you'll first need [Bazel](https://bazel.build/) installed on your machine (version 1.0.0+ is *required*).

To build the vdisc CLI, you'll need to clone this git repository and build the vdisc command with Bazel.

```sh
$ git clone https://github.com/NVIDIA/vdisc.git
$ cd vdisc
$ bazel build --stamp //cmd/vdisc
```

Getting Started
---------------

Burning a vdisc using the CLI is simple. First you need to generate a manifest of your objects, where they should appear in the disc image, and their size.


```sh
$ cat << EOF > mnist.csv
"/train-images-idx3-ubyte.gz","https://storage.googleapis.com/tensorflow/tf-keras-datasets/train-images-idx3-ubyte.gz",9912422
"/train-labels-idx1-ubyte.gz","https://storage.googleapis.com/tensorflow/tf-keras-datasets/train-labels-idx1-ubyte.gz",28881
"/t10k-images-idx3-ubyte.gz","https://storage.googleapis.com/tensorflow/tf-keras-datasets/t10k-images-idx3-ubyte.gz",1648877
"/t10k-labels-idx1-ubyte.gz","https://storage.googleapis.com/tensorflow/tf-keras-datasets/t10k-labels-idx1-ubyte.gz",4542
EOF
$ vdisc burn -i mnist.csv -o mnist.vdsc
```

Once you've burned a vdisc, you can mount it

```
$ mkdir mnist
$ vdisc mount --url=mnist.vdsc --mountpoint=mnist
1.570744835687179e+09	info	maxprocs/maxprocs.go:47	maxprocs: Leaving GOMAXPROCS=8: CPU quota undefined
1.570744835729569e+09	info	isofuse/isofuse.go:68	mounted iso	{"mountpoint": "/home/joeuser/mnist"}
```

And in another terminal you can examine the files

```
sh
$ md5 mnist/*
MD5 (mnist/t10k-images-idx3-ubyte.gz) = 074392edd37ac2bd4904c0df2a31c38e
MD5 (mnist/t10k-labels-idx1-ubyte.gz) = dbb5c5e00e8b64dfe161442e122f1c8b
MD5 (mnist/train-images-idx3-ubyte.gz) = b1c2f15e5ea102012fa9da59cd0d6d7c
MD5 (mnist/train-labels-idx1-ubyte.gz) = e538dc41040b558f796a632c4604bbeb
```

By default, vdisc mount uses fuse, but on linux you can TCMU by specifying `--mode=tcmu`.

Architecture
------------

To learn more about how vdisc works, read through the [detailed design](/doc/design.md).

#  VDisc Design

## Background

While developing solutions for traceable, reproducible and safe AI-powered applications, NVIDIA has developed an immutable file system solution for datasets called VDiscs. Here we discuss the origin of VDiscs and their design.

## Read-Only Volumes

To meet our goals to provide reproducibility and traceability in training models, we wish to have read-only snapshots of our datasets. Additionally, we want to be able to access these datasets from a wide range of tools. Many of these tools already have a heavy reliance on file system access to data. These constraints focused our attention on existing read-only, POSIX file systems for Linux. There are several read-only filesystems supported by the Linux kernel including iso9660 (the CD ROM file system), squashfs, and cramfs. We initially focused on iso9660 because of its broad platform support.

Let's imagine we wanted to create an iso disk image to snapshot our dataset. On Linux we run

    genisoimage -r -o /path/to/mnist.iso /my/datasets/mnist
    losetup /dev/loop1 /path/to/mnist.iso
    mount -t iso9660 /dev/loop1 /mnt/mnist

This generates an iso image, attaches it to a loopback block device, and then mounts the dataset on `/mnt/mnist`. Cool! Now we have an immutable dataset that we can mount on Linux or macOS or Windows even.

## Object Storage

Now that we have an immutable image of our dataset, the question is "How do we share our image across a large compute cluster?". If we are doing anything interesting at all our dataset is quite large, and it would be costly to download the entire thing to every compute node. Wouldn't it be cool if we could store the iso image on remote storage and have our loopback device read blocks from there?

Unfortunately, the `losetup` command doesn't support passing an HTTP URL for the image location. Luckily there are some options though. Linux implements a few different technologies that allow you to simulate a block device. During the implementation of vdiscs, we focused on two of these: NBD and TCMU. NBD stands for Network Block Device and TCMU is an iSCSI in user space target.

Our initial implementation of an losetup for remote objects was an NBD server bound to localhost that translated `NBD_CMD_READ` requests into HTTP Range requests. That was it. We upload our iso image to S3, and then create a network block device on localhost. Whenever the kernel wants to read a block (or serveral) from the iso image, the server makes an HTTP Range request to S3 for the slice of our iso image.

    genisoimage -o /path/to/dataset.iso /my/dataset/directory
    aws s3 cp /path/to/dataset.iso s3://path/to/dataset.iso
    nbd_losetup /dev/nbd1 s3://path/to/dataset.iso
    mount -t iso9660 /dev/nbd1 /mnt/dataset

We wound up switching from NBD to TCMU because TCMU has better throughput and it supports transparent restarts of the server. The idea is nearly identical though. Instead of handling an `NBD_CMD_READ` request we pull a SCSI `Read6/Read10/Read12/Read16` command off of the TCMU ring buffer and make an HTTP Range request.

    genisoimage -o /path/to/dataset.iso /my/dataset/directory
    aws s3 cp /path/to/dataset.iso s3://path/to/dataset.iso
    tcmu_losetup /dev/tcmu1 s3://path/to/dataset.iso
    mount -t iso9660 /dev/tcmu1 /mnt/dataset

## Data Sharing

The next big idea we wanted to explore was deduplication of data between datasets. Imagine you have 2 7TB datasets with 95% overlapping content. With our current solution, we need to generate two 7TB iso images, and upload them to S3.

### ISO9660 File System Layout

The ISO 9660 file system is an extent based file system with the following file layout

    +-------------+
    |   Header    |\
    +-------------+ \
    | Path Table  |  \
    +-------------+   \
    | Directory 1 |    \
    +-------------+     +- Metadata
    | Directory 2 |    /
    +-------------+   /
    | ...         |  /
    +-------------+ /
    | Directory N |/
    +-------------+
    |   File 1    |\
    +-------------+ \
    |   File 2    |  \
    +-------------+   +--- Data
    | ...         |  /
    +-------------+ /
    |   File M    |/
    +-------------+

The metadata portion of the file system dictates the layout of all the files when the iso is mounted. Each file entry in a directory includes the the extent where the file content is stored in the data segments.

Wouldn't it be cool if we store each unique file in our object store and then virtualize the data segments?

### VDisc Structure

You may have noticed that the tcmu_losetup program doesn't actually know anything about the file system contained in the image it is providing. It is a dump pass-through block device simulator. Each Read request is converted into an HTTP Range request for to a slice of a single object in our object store. But what if we make it slightly fancier and divide our block device into multiple backing files? A vdisc is just a serialized structure which keeps the mapping of which objects correspond to which range of blocks to make up our block device.

Here is the cap'n proto definition of a vdisc

    struct VDisc {
      blockSize @0 :UInt16;
      extents @2 :List(Extent);
    }
    
    struct Extent {
      uri @0 :Text;
      num_blocks @1 :UInt32;
      padding @2 :UInt16;
    }

As an example you could split an iso image into two objects, one for metadata and the other for data. The vdisc would contain two extents e.g.,

    {
      "blockSize": 2048,
      "extents": [
        {
          "uri": "s3://mybucket/dataset.iso.metadata",
          "num_blocks": 104,
          "padding": 0,
        },
        {
          "uri": "s3://mybucket/dataset.iso.data",
          "num_blocks": 3423407,
          "padding": 0,
        }
      ]
    }

More usefully, we can map every object in our dataset to a separate extent in a vdisc e.g.,

    {
      "blockSize": 2048,
      "extents": [
        {
          "uri": "s3://mybucket/dataset.iso.metadata",
          "num_blocks": 104,
          "padding": 0,
        },
        {
          "uri": "s3://mybucket/objects/123.jpg",
          "num_blocks": 10,
          "padding": 0,
        },
        {
          "uri": "s3://mybucket/objects/34534.jpg",
          "num_blocks": 10,
          "padding": 0,
        },
        ...
        {
          "uri": "s3://mybucket/objects/blah.tfrecords",
          "num_blocks": 10,
          "padding": 0,
        }
      ]
    }

To create one of these vdisc structures (along with the iso metadata file) we can use the `vdisc create` command. As input it takes a CSV file which contains a line for every file to be added to the iso image. For example, the csv for an mnist vdisc might contain

    "/t10k-images-idx3-ubyte.gz","s3://mybucket/mnist/t10k-images-idx3-ubyte.gz","1648877"
    "/t10k-labels-idx1-ubyte.gz","s3://mybucket/mnist/t10k-labels-idx1-ubyte.gz","4542"
    "/train-images-idx3-ubyte.gz","s3://mybucket/mnist/train-images-idx3-ubyte.gz","9912422"
    "/train-labels-idx1-ubyte.gz","s3://mybucket/mnist/train-labels-idx1-ubyte.gz","28881"

The columns in the csv are iso\_path, object\_url, object\_size, and object\_checksum. To create the vdisc with an iso file system for this input you'd run

    vdisc burn -i mnist.csv -o s3://mybucket/mnist.vdsc

This command reads the CSV file, generates the iso metadata object, uploads it to S3 and then generates a vdisc containing roughly

    {
      "fsType": "iso9660",
      "blockSize": 2048,
      "extents": [
        {
          "uri": "s3://mybucket/mnist.vdsc.isohdr",
          "num_blocks": 104,
          "padding": 0,
        },
        {
          "uri": "s3://mybucket/mnist/t10k-images-idx3-ubyte.gz",
          "num_blocks": 805,
          "padding": 1811,
        },
        {
          "uri": "s3://mybucket/mnist/t10k-labels-idx1-ubyte.gz",
          "num_blocks": 2,
          "padding": 1602,
        },
        {
          "uri": "s3://mybucket/mnist/train-images-idx3-ubyte.gz",
          "num_blocks": 4840,
          "padding": 1946,
        },
        {
          "uri": "s3://mybucket/mnist/train-labels-idx1-ubyte.gz",
          "num_blocks": 14,
          "padding": 1839,
        }
      ]
    }

and ultimately the vdisc structure is serialize using cap'n proto, gzipped, and uploaded to s3://mybucket/mnist.vdisc.

### VDisc Mounting

Now that we have this cool vdisc structure mapping objects to extents of our block device we can modify our tcmu_losetup program to download a vdisc structure issue HTTP Range requests to the appropriate object(s) based on the blocks requested in the SCSI Read* command. Our example becomes

    aws s3 cp -r /my/datasets/mnist s3://mybucket/mnist
    ... # make an mnist.csv
    vdisc burn -i mnist.csv -o s3://mybucket/mnist.vdsc
    vdisc mount --mode=tcmu -u s3://mybucket/mnist.vdsc -p /mnt/mnist

This downloads the vdisc, loads the extent mapping, creates a TCMU device backed by the virtualized extents, and then mounts the device at /mnt/mnist.

When we make a POSIX Open() call on a file like /mnt/mnist/train-images-idx3-ubyte.gz the kernel issues block IOs for the blocks mapping to s3://mybucket/mnist.vdsc.isohdr to check if the file exists, check permissions, and find the extent containing the file. Once the file is open and a POSIX Read() is issued, the vdisc TCMU server issues the read calls to the s3://mybucket/mnist/train-images-idx3-ubyte.gz as HTTP Range request.

The flow of I/O requests looks like

    +---------------------------------------+
    |                                       |     HTTP
    |                  Job         vdiscd --^-------------> S3/Swift
    |                   |            ^      |
    | ring3             |            |      |
    |---------------------------------------|
    | ring0             |            |      |
    |                   |            |      |
    |     Page Cache - VFS           |      |
    |                   |            |      |
    |      FS-Cache - ISOFS          |      |
    |                   |            |      |
    |                BLK layer       |      |
    |                   |            |      |
    |               TCM target       |      |
    |                   |            |      |
    |                TCM loop -------+      |
    |                                       |
    +---------------------------------------+

## Local Caching

Because vdiscs are implemented as POSIX file systems on Linux, we are able to take advantage of several different compute-local caching solutions.

### Page Cache

The simplest form of local caching we get for free. Because vdiscs are mounted as a regular Linux file system, we take advantage of the Linux page cache. When a process reads from a file on Linux, the virtual file system (vfs) layer automatically uses unused pages of memory as a cache. Subsequent reads of the same page of a file will be read directly from RAM.

### Fs-Cache

TODO

### Read-Ahead

TODO

## TCMU Ring Recovery

TODO

## ISO 9660 Limitations

ISO 9660 filesystems can have up to 2^32 blocks, i.e. 8 TiB. The maximum size of data files depends on the Level of Interchange that is intended for the ISO filesystem. VDisc implements Level 3 which allows to have multiple consequtive Directory Records with the same name. They all are to be concatenated to a single data file. This means that a single data file can nearly fill up the full 8 TiB of image size.

## FUSE

TODO

## Block Device Performance Tuning

TODO

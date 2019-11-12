# VDisc Storage

## Background

VDisc supports reading file data from different URL schemes. These are
documented below.

## data

`data:[<media type>][;base64],<data>`

The data URL scheme is described at https://en.wikipedia.org/wiki/Data_URI_scheme

## file

`file:///<path/to/local/file>`

TODO

## http

`https://<endpoint>/<object>`

TODO

## S3

`s3://<bucket>/<object>`

TODO

### Auth considerations

TODO

## Swift / s3api

`swift://<endpoint>/<owner>/<container>/<object>`

When talking to a Swift cluster, use the `swift` URL schema. Internally,
this schema uses the S3 API to talk to the Swift cluster, so the Swift
cluster needs to have enabled `s3api` support.

The URL schema has four key parts:

`<endpoint>`
:   Server hostname, including port designation, if needed

`<owner>`
:   The owner of the container and object being referenced. In practice,
    this normally ends up being the same value as the `SWIFT_ACCESS_KEY_ID`

`<container>`
:   The name of the Swift container being used. This is analagous to an S3
    bucket.

`<object>`
:   The name of the object. The object name may include the `/` character.

### Auth considerations

VDisc uses the following environment variables for `swift` auth:

    SWIFT_ACCESS_KEY_ID=user
    SWIFT_SECRET_ACCESS_KEY=s3api password

By default, VDisc uses the `us-east-1` region. If your Swift cluster is
configured with a different s3api region, set the correct value in the
`SWIFT_REGION` environment variable.

## zero

`zero:<length>`

Returns a byte stream with a fixed length. eg `zero:72` is a stream of
72 null bytes. This is useful for testing.

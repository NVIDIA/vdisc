#!/bin/bash

# We'll remove these exclusions when we have time to go back and fix older code.
# See golint commonInitialisms - https://github.com/golang/lint/blob/8f45f776aaf18cebc8d65861cc70c33c60471952/lint.go#L771
./bazel-bin/external/org_golang_x_lint/golint/$(uname -s | tr '[:upper:]' '[:lower:]')_amd64_stripped/golint -- $1  | grep -v "should have comment" | grep -E -v "should be .*(ACL|API|ASCII|CPU|CSS|DNS|EOF|GUID|HTML|HTTP|HTTPS|ID|IP|JSON|LHS|QPS|RAM|RHS|RPC|SLA|SMTP|SQL|SSH|TCP|TLS|TTL|UDP|UI|UID|UUID|URI|URL|UTF8|VM|XML|XMPP|XSRF|XSS)" | grep -v "CamelCase" | grep -v "package comment" | grep -v "don't use underscores in Go names" | grep -v "don't use an underscore in package name" | grep -v "comment on exported" | grep -v "consider calling this" | (! grep -E ".+")

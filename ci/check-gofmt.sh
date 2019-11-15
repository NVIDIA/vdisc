#!/bin/sh
# Copyright (c) 2010, Evan Shaw
# All rights reserved.

# Redistribution and use in source and binary forms, with or without
# modification, are permitted provided that the following conditions are met:
#     * Redistributions of source code must retain the above copyright
#       notice, this list of conditions and the following disclaimer.
#     * Redistributions in binary form must reproduce the above copyright
#       notice, this list of conditions and the following disclaimer in the
#       documentation and/or other materials provided with the distribution.
#     * Neither the name of the copyright holder nor the
#       names of its contributors may be used to endorse or promote products
#       derived from this software without specific prior written permission.

# THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
# ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
# WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
# DISCLAIMED. IN NO EVENT SHALL <COPYRIGHT HOLDER> BE LIABLE FOR ANY
# DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
# (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
# LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
# ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
# (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
# SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

bazel build @go_sdk//...

IFS='
'
for file in `git diff --name-only --diff-filter=ACM HEAD~1 | grep '\.go$'`
do
    output=`git cat-file -p :$file | ./bazel-vdisc*/external/go_sdk/bin/gofmt -l 2>&1`
    if test $? -ne 0
    then
        output=`echo "$output" | sed "s,<standard input>,$file,"`
        syntaxerrors="${list}${output}\n"
    elif test -n "$output"
    then
        list="${list}${file}\n"
    fi
done
exitcode=0
if test -n "$syntaxerrors"
then
    echo >&2 "gofmt found syntax errors:"
    printf "$syntaxerrors"
    exitcode=1
fi
if test -n "$list"
then
    echo >&2 "gofmt needs to format these files (run gofmt -w and git add):"
    printf "$list"
    exitcode=1
fi
exit $exitcode

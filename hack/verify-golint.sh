#!/bin/bash

# Copyright (C) 2018 Synopsys, Inc.
#
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements. See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership. The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License. You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied. See the License for the
# specific language governing permissions and limitations
# under the License.

ROOT=$(dirname "${BASH_SOURCE}")/..

cd "${ROOT}"

if ! which golint > /dev/null; then
  echo 'Can not find golint, installing with:'
  echo 'go get -u github.com/golang/lint/golint'
  go get -u github.com/golang/lint/golint
fi

packages=(
  $(go list -e ./...)
)

errors=()
for p in "${packages[@]}"
do
  result=$(golint "$p")
  if [[ -n "${result}" ]]
  then
    errors+=( "${result}" )
  fi
done

if [ ${#errors[@]} -eq 0 ]; then
  echo 'All Go source files passed lint checks.'
else
  {
    echo "Errors from golint:"
    for err in "${errors[@]}"
    do
      echo "$err"
    done
  } >&2
  exit 1
fi

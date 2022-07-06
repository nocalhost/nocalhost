#!/usr/bin/env bash

#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

RELEASE=${RELEASE:-$2}
PREVIOUS_RELEASE=${PREVIOUS_RELEASE:-$1}

## Ensure Correct Usage
if [[ -z "${PREVIOUS_RELEASE}" || -z "${RELEASE}" ]]; then
  echo Usage:
  echo ./scripts/release/release-notes.sh v3.0.0 v3.1.0
  echo or
  echo PREVIOUS_RELEASE=v3.0.0
  echo RELEASE=v3.1.0
  echo ./scripts/release/release-notes.sh
  exit 1
fi

## validate git tags
for tag in $RELEASE $PREVIOUS_RELEASE; do
  OK=$(git tag -l ${tag} | wc -l)
  if [[ "$OK" == "0" ]]; then
    echo ${tag} is not a valid release version
    exit 1
  fi
done

## Generate CHANGELOG from git log
CHANGELOG=$(git log --no-merges --pretty=format:'- %s %H (%aN)' ${PREVIOUS_RELEASE}..${RELEASE})
if [[ ! $? -eq 0 ]]; then
  echo "Error creating changelog"
  echo "try running \`git log --no-merges --pretty=format:'- %s %H (%aN)' ${PREVIOUS_RELEASE}..${RELEASE}\`"
  exit 1
fi

## guess at MAJOR / MINOR / PATCH versions
MAJOR=$(echo ${RELEASE} | sed 's/^v//' | cut -f1 -d.)
MINOR=$(echo ${RELEASE} | sed 's/^v//' | cut -f2 -d.)
PATCH=$(echo ${RELEASE} | sed 's/^v//' | cut -f3 -d.)

## Print release notes to stdout
cat <<EOF
## ${RELEASE}
Nocalhost ${RELEASE} is a feature release.
## Installation and Upgrading
Please follow [Install Nocalhost](https://nocalhost.dev/docs/installation) to install Nocalhost IDE plug-in. Upgrading the IDE plug-in will automatically upgrade nhctl.
## Changelog
${CHANGELOG}
EOF
#!/bin/bash

# Copyright 2020 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script will generate a table of contents (ToC) for Markdown (MD) files.
#
#  1. Remove the paragraphs (headings) above the "Table of Contents"
#  2. Remove code blocks fenced by tripple back-ticks, to not treat #-comments as markdown headings
#  3. Find the paragraph headings with grep (1st through 4th level heading starting with "#" and "####")
#  4. Extract the heading's text with sed and transform into '|'-separated records of the form '###|Full Text|Full Text'
#  5. Generate the ToC lines with awk by replacing '#' with '  ', converting spaces to dashes '-',
#     removing special chars (like back-ticks, dot, parenthesis, colon, comma) from TOC anchor links,
#     and lower-case all capital letters
#  6. Remove leading 2 spaces in case ToC does not include 1st level headings, otherwise the TOC becomes a code block
#
# Inspired by https://medium.com/@acrodriguez/one-liner-to-generate-a-markdown-toc-f5292112fd14

SEP="|"

[ -z "${1}" ] && echo -e "Usage:\n\n   $BASH_SOURCE <markdown file>\n" && exit 1

sed -n '/Table of Contents/,$p' "${1}" | tail -n +2 | \
sed '/^```/,/^```/d' | \
grep -E "^#{1,4}" | \
sed -E "s/(#+) (.+)/\1${SEP}\2${SEP}\2/g" | \
awk -F "${SEP}" '{ gsub(/#/,"  ",$1); gsub(/[ ]/,"-",$3); gsub(/[`.():,]/,"",$3); print $1 "- [" $2 "](#" tolower($3) ")" }' | \
sed -e 's/^  //g'

#!/bin/bash

url=$1
secret=$2
payload=$3

signature=$(echo -n "$payload" | openssl dgst -sha256 -hmac "$secret" | cut -d ' ' -f2)

curl -X POST ${url} -H "X-Hub-Signature-256: ${signature}" --data "$payload"
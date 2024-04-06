#!/bin/sh
npm i

GO_POST_PROCESS_FILE="/usr/local/bin/gofmt -w" npx openapi-generator-cli generate \
  --skip-validate-spec -g go  \
  --additional-properties="packageName=client" \
  -o client -i http://localhost:8000/rpc/swagger.json

# cleanup
rm -r client/test
rm -r client/docs
rm -r client/go.*
rm -r client/README.md
rm -r client/git_push.sh
rm -r client/.travis.yml
rm -r client/.openapi-generator-ignore
rm -r client/.gitignore
rm -r client/api
rm -rf client/.openapi-generator
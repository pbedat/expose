#!/bin/sh
cd client 
npm i
npx openapi-typescript http://localhost:8000/rpc/swagger.json --empty-objects-unknown -o client/api.d.ts
npx tsc

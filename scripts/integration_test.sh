#!/usr/bin/env sh

# get Go package list from caller
PACKAGES=$1

# start the mountebank container at localhost:2525, with ports 8080 and 8081 for test imposters
docker run -d --rm --name=mountebank_test -p 2525:2525 -p 8080:8080 -p 8081:8081 andyrbell/mountebank:2.0.0

# run integration tests and record exit code
go test -cover -tags=integration -timeout=5s ${PACKAGES}
CODE=$?

# always stop the mountebank container, even on failures
docker stop mountebank_test

exit ${CODE}
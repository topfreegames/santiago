#!/bin/bash

VERSION=$(cat ./metadata/version.go | grep "var VERSION" | awk ' { print $4 } ' | sed s/\"//g)

cp ./config/default.yaml ./dev

docker build -t santiago .
docker build -t santiago-worker -f ./WorkerDockerfile .
docker build -t santiago-dev -f ./DevDockerfile .
docker login -e="$DOCKER_EMAIL" -u="$DOCKER_USERNAME" -p="$DOCKER_PASSWORD"

docker tag santiago:latest tfgco/santiago:$VERSION.$TRAVIS_BUILD_NUMBER
docker push tfgco/santiago:$VERSION.$TRAVIS_BUILD_NUMBER
docker tag santiago:latest tfgco/santiago:$VERSION
docker push tfgco/santiago:$VERSION
docker tag santiago:latest tfgco/santiago:latest
docker push tfgco/santiago

docker tag santiago-worker:latest tfgco/santiago-worker:$VERSION.$TRAVIS_BUILD_NUMBER
docker push tfgco/santiago-worker:$VERSION.$TRAVIS_BUILD_NUMBER
docker tag santiago-worker:latest tfgco/santiago-worker:$VERSION
docker push tfgco/santiago-worker:$VERSION
docker tag santiago-worker:latest tfgco/santiago-worker:latest
docker push tfgco/santiago-worker

docker tag santiago-dev:latest tfgco/santiago-dev:$VERSION.$TRAVIS_BUILD_NUMBER
docker push tfgco/santiago-dev:$VERSION.$TRAVIS_BUILD_NUMBER
docker tag santiago-dev:latest tfgco/santiago-dev:$VERSION
docker push tfgco/santiago-dev:$VERSION
docker tag santiago-dev:latest tfgco/santiago-dev:latest
docker push tfgco/santiago-dev

DOCKERHUB_LATEST=$(python ./scripts/get_latest_tag.py)

if [ "$DOCKERHUB_LATEST" != "$VERSION.$TRAVIS_BUILD_NUMBER" ]; then
    echo "Last version is not in docker hub!"
    echo "docker hub: $DOCKERHUB_LATEST, expected: $VERSION.$TRAVIS_BUILD_NUMBER"
    exit 1
fi

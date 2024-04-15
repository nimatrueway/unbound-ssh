#!/usr/bin/env bash
# release script
SCRIPT_DIR=$(cd $(dirname $0); pwd)

if git diff --quiet
then
  :
else
  git status -s
  echo -e "\nThere are uncommitted changes, "
  read -p "would you like to continue? " -n 1 -r
fi

# build
"$SCRIPT_DIR"/build.sh

# figure out the new version and changelog
CHANGELOG_LAST_VERSION=$(perl -nE 'say $1 if /## (.*)/' ./CHANGELOG.md | head -n 1)
LATEST_TAG=$(git describe --tags --abbrev=0)
CHANGELOG_2ND_LAST_VERSION=$(perl -nE 'say $1 if /## (.*)/' ./CHANGELOG.md | head -n 2 | tail -n 1)
if [ "$CHANGELOG_2ND_LAST_VERSION" == "$LATEST_TAG" ]; then
  echo "CHANGELOG.md is not updated for the latest tag"
  exit 1
fi
NEW_CHANGELOG=$(awk "/^## $CHANGELOG_LAST_VERSION$/{flag=1;next}/^## $CHANGELOG_2ND_LAST_VERSION$/{flag=0}flag" ./CHANGELOG.md)
echo -e "\n\n$CHANGELOG_LAST_VERSION\n$NEW_CHANGELOG\n\n"
read -p "Does the changelog look good? (y/n) " -n 1 -r

# add tag and push it
git tag "$CHANGELOG_LAST_VERSION"
git push origin "$CHANGELOG_LAST_VERSION"

# create github release
gh release create "$CHANGELOG_LAST_VERSION" --title "$CHANGELOG_LAST_VERSION" --notes-file <(echo "$NEW_CHANGELOG") ./output/*

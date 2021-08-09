echo "Publishing the coverage results"
if [ "$TRAVIS_GO_VERSION" == "tip" ]; then
	echo "Coverage information is not required for tip version."
	exit 0
fi

mkdir -p $TRAVIS_BUILD_DIR/gh-pages
cd $TRAVIS_BUILD_DIR/gh-pages

OLD_COVERAGE=0
NEW_COVERAGE=0
RESULT_MESSAGE=""

BADGE_COLOR=red
GREEN_THRESHOLD=85
YELLOW_THRESHOLD=50

# clone and prepare gh-pages branch
git clone -b gh-pages https://$GHE_USER:$GHE_TOKEN@github.com/$TRAVIS_REPO_SLUG.git .
git config user.name "travis"
git config user.email "travis"

if [ ! -d "$TRAVIS_BUILD_DIR/gh-pages/coverage" ]; then
	mkdir "$TRAVIS_BUILD_DIR/gh-pages/coverage"
fi

if [ ! -d "$TRAVIS_BUILD_DIR/gh-pages/coverage/$TRAVIS_BRANCH" ]; then
	mkdir "$TRAVIS_BUILD_DIR/gh-pages/coverage/$TRAVIS_BRANCH"
fi

if [ ! -d "$TRAVIS_BUILD_DIR/gh-pages/coverage/$TRAVIS_COMMIT" ]; then
	mkdir "$TRAVIS_BUILD_DIR/gh-pages/coverage/$TRAVIS_COMMIT"
fi

if [ -f "$GOPATH/src/github.com/kubernetes-sigs/ibm-vpc-block-csi-driver/Passing" ]; then
	curl https://img.shields.io/badge/e2e-passing-Yellow.svg > $TRAVIS_BUILD_DIR/gh-pages/coverage/$TRAVIS_BRANCH/e2e.svg
elif [ -f "$GOPATH/src/github.com/kubernetes-sigs/ibm-vpc-block-csi-driver/Failed" ]; then
	curl https://img.shields.io/badge/e2e-failed-Yellow.svg > $TRAVIS_BUILD_DIR/gh-pages/coverage/$TRAVIS_BRANCH/e2e.svg
fi

# Compute overall coverage percentage
echo "Computing the coverages"
OLD_COVERAGE=$(cat $TRAVIS_BUILD_DIR/gh-pages/coverage/$TRAVIS_BRANCH/cover.html  | grep "%)"  | sed 's/[][()><%]/ /g' | awk '{ print $4 }' | awk '{s+=$1}END{print s/NR}')
cp $TRAVIS_BUILD_DIR/cover.html $TRAVIS_BUILD_DIR/gh-pages/coverage/$TRAVIS_BRANCH
cp $TRAVIS_BUILD_DIR/cover.html $TRAVIS_BUILD_DIR/gh-pages/coverage/$TRAVIS_COMMIT
NEW_COVERAGE=$(cat $TRAVIS_BUILD_DIR/gh-pages/coverage/$TRAVIS_BRANCH/cover.html  | grep "%)"  | sed 's/[][()><%]/ /g' | awk '{ print $4 }' | awk '{s+=$1}END{print s/NR}')

if (( $(echo "$NEW_COVERAGE > $GREEN_THRESHOLD" | bc -l) )); then
	BADGE_COLOR="green"
elif (( $(echo "$NEW_COVERAGE > $YELLOW_THRESHOLD" | bc -l) )); then
	BADGE_COLOR="yellow"
fi

# Generate badge for coverage
curl https://img.shields.io/badge/coverage-$NEW_COVERAGE-$BADGE_COLOR.svg > $TRAVIS_BUILD_DIR/gh-pages/coverage/$TRAVIS_BRANCH/badge.svg

COMMIT_RANGE=(${TRAVIS_COMMIT_RANGE//.../ })

# Generate result message for log and PR
if (( $(echo "$OLD_COVERAGE > $NEW_COVERAGE" | bc -l) )); then
	RESULT_MESSAGE=":red_circle: Coverage decreased from [$OLD_COVERAGE%](https://pages.github.com/$TRAVIS_REPO_SLUG/coverage/${COMMIT_RANGE[0]}/cover.html) to [$NEW_COVERAGE%](https://pages.github.com/$TRAVIS_REPO_SLUG/coverage/${COMMIT_RANGE[1]}/cover.html)"
elif (( $(echo "$OLD_COVERAGE == $NEW_COVERAGE" | bc -l) )); then
	RESULT_MESSAGE=":thumbsup: Coverage remained same at [$NEW_COVERAGE%](https://pages.github.com/$TRAVIS_REPO_SLUG/coverage/${COMMIT_RANGE[1]}/cover.html)"
else
	RESULT_MESSAGE=":thumbsup: Coverage increased from [$OLD_COVERAGE%](https://pages.github.com/$TRAVIS_REPO_SLUG/coverage/${COMMIT_RANGE[0]}/cover.html) to [$NEW_COVERAGE%](https://pages.github.com/$TRAVIS_REPO_SLUG/coverage/${COMMIT_RANGE[1]}/cover.html)"
fi

# Update gh-pages branch or PR
echo "Updating gh-pages"
if [ "$TRAVIS_PULL_REQUEST" == "false" ]; then
	git status
	git add .
	git commit -m "Coverage result for commit $TRAVIS_COMMIT from build $TRAVIS_BUILD_NUMBER"
	git push origin
else
        # Updates PR with coverage
        curl -i -H "Authorization: token $GHE_API_TOKEN" https://github.com/api/v3/repos/$TRAVIS_REPO_SLUG/issues/$TRAVIS_PULL_REQUEST/comments --data '{"body": "'"$RESULT_MESSAGE"'"}'
fi

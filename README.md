Original plugin (https://github.com/drone/drone-jira) extracts only 1 issue number for Jira Deployment. I changed the plugin to extract multiple issues (if they exist) which is usedful when you use ```git merge --squash``` to merge multiple commits as one. All issue numbers are then sent to Jira Deployment (instead of only 1 as in the original plugin).

---
A plugin to attach build and deployment details to a Jira issue.

# Building

Build the plugin binary:

```text
scripts/build.sh
```

Build the plugin image:

```text
docker build -t plugins/jira -f docker/Dockerfile .
```

# Testing

Execute the plugin from your current working directory:

```text
docker run --rm \
  -e DRONE_COMMIT_SHA=8f51ad7884c5eb69c11d260a31da7a745e6b78e2 \
  -e DRONE_COMMIT_BRANCH=master \
  -e DRONE_COMMIT_AUTHOR=octocat \
  -e DRONE_COMMIT_AUTHOR_EMAIL=octocat@github.com \
  -e DRONE_COMMIT_MESSAGE="DRONE-42 updated the readme" \
  -e DRONE_BUILD_NUMBER=43 \
  -e DRONE_BUILD_STATUS=success \
  -e DRONE_BUILD_LINK=https://cloud.drone.io \
  -e PLUGIN_CLOUD_ID=${JIRA_CLOUD_ID} \
  -e PLUGIN_CLIENT_ID=${JIRA_CLIENT_ID} \
  -e PLUGIN_CLIENT_SECRET=${JIRA_CLIENT_SECRET} \
  -e PLUGIN_PROJECT=${JIRA_PROJECT} \
  -e PLUGIN_PIPELINE=drone \
  -e PLUGIN_ENVIRONMENT=production \
  -e PLUGIN_STATE=successful \
  -w /drone/src \
  -v $(pwd):/drone/src \
  plugins/jira
```

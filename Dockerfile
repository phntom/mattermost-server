# go mod tidy
# BUILD_ENTERPRISE_READY="false" BUILD_VERSION="7.7" BUILD_NUMBER="7.7.0" BUILD_DATE="Sun 22 Jan 2023 12:16:50 IST" BUILD_HASH="b57aa4cf1f0b8acba0fb475a76928bff11b377a0" CGO_ENABLED=0 make build-linux
# cp -rv ../mattermost-webapp/dist client

FROM mattermost/mattermost-team-edition:7.7.0

USER root
RUN apt update && apt install -y libc6 libc-bin && apt clean
USER mattermost

COPY --chown=2000:2000 bin/mattermost /mattermost/bin/
COPY --chown=2000:2000 client/* /mattermost/client/
COPY --chown=2000:2000 com.mattermost.plugin-todo-0.7.0.tar.gz* /mattermost/prepackaged_plugins/

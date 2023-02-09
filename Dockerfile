# go mod tidy
# BUILD_ENTERPRISE_READY="false" BUILD_VERSION="7.7" BUILD_NUMBER="7.7.1" BUILD_DATE="Thu 09 Feb 2023 09:13:29 IST" BUILD_HASH="f4990e8849e50b064c5d6d248bdaf507c0db827c" CGO_ENABLED=0 make build-linux
# cp -rv ../mattermost-webapp/dist client

FROM mattermost/mattermost-team-edition:7.7.1

USER root
RUN apt update && apt install -y libc6 libc-bin && apt clean
USER mattermost

COPY --chown=2000:2000 bin/mattermost /mattermost/bin/
COPY --chown=2000:2000 client/* /mattermost/client/
COPY --chown=2000:2000 com.mattermost.plugin-todo-0.7.0.tar.gz* /mattermost/prepackaged_plugins/

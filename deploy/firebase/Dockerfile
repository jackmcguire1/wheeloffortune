# Firebase emulator suite for local development. Needs Java for the emulators.
FROM node:24-alpine
RUN apk add --no-cache openjdk25-jre-headless bash \
 && npm install -g firebase-tools@15
WORKDIR /firebase
COPY firebase.json .firebaserc ./
EXPOSE 4000 9099
CMD ["firebase", "emulators:start", "--only", "auth", "--project", "demo-wheeloffortune"]

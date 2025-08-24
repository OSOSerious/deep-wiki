FROM openjdk:21-jdk-slim AS build
WORKDIR /workspace
COPY . .
RUN ./gradlew bootJar --no-daemon

FROM openjdk:21-jdk-slim
WORKDIR /app
COPY --from=build /workspace/build/libs/user-0.0.1-SNAPSHOT.jar app.jar
EXPOSE 8080
ENTRYPOINT ["java","-jar","app.jar"]
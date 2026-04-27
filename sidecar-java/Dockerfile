# ---- Stage 1: Build — Maven compile + package ----
FROM maven:3.9-eclipse-temurin-21 AS builder
WORKDIR /app
COPY pom.xml .
RUN mvn dependency:go-offline
COPY src ./src
RUN mvn package -DskipTests

# ---- Stage 2: Runtime — minimal JRE image ----
FROM eclipse-temurin:21-jre
WORKDIR /app
COPY --from=builder /app/target/sidecar.jar sidecar.jar
RUN mkdir -p /run/armorclaw
USER 10001:10001
ENTRYPOINT ["java", "-jar", "sidecar.jar"]

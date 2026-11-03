FROM maven:3.9-eclipse-temurin-21 AS build
WORKDIR /app
COPY . .
RUN ./mvnw -B package -DskipTests

FROM eclipse-temurin:21-jre
COPY --from=build /app/target/app.jar /app.jar
CMD ["java", "-jar", "/app.jar"]

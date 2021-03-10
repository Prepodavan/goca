FROM swaggerapi/swagger-codegen-cli:latest

RUN wget \
    https://repo1.maven.org/maven2/io/swagger/codegen/v3/swagger-codegen-cli/3.0.20/swagger-codegen-cli-3.0.20.jar \
    -O /main.jar

COPY build/swagger-entrypoint.sh swagger-entrypoint.sh

ENTRYPOINT ["sh", "swagger-entrypoint.sh"]

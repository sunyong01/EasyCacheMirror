## 测试Maven
```bash
docker run -it --rm maven:3.8-openjdk-11 bash -c "mkdir -p /root/.m2 && \
echo '<settings>
    <mirrors>
        <mirror>
            <id>my-mirror</id>
            <mirrorOf>central</mirrorOf>
            <url>http://192.168.0.124:8080/maven</url>
        </mirror>
    </mirrors>
</settings>' > /root/.m2/settings.xml && \
mvn dependency:get -U -Dartifact=com.google.guava:guava:31.1-jre"
```


## 测试Gradle
```bash
docker run -it --rm gradle:jdk21-corretto bash -c "
mkdir -p /home/gradle/project && \
echo 'plugins {
    id \"java\" // 应用 Java 插件
    id \"org.springframework.boot\" version \"3.1.4\" // 应用 Spring Boot 插件 
    id \"io.spring.dependency-management\" version \"1.1.3\" // 管理依赖版本
}

repositories {
    maven {
        url \"http://192.168.0.124:8080/maven/\"
        allowInsecureProtocol = true
    }
    mavenCentral()
}

dependencies {
    implementation \"com.google.guava:guava:31.1-jre\"
    implementation \"org.springframework.boot:spring-boot-starter-aop\"
    implementation \"org.springframework.boot:spring-boot-starter-security\"
    implementation \"org.springframework.boot:spring-boot-starter-web\"
    implementation \"org.springframework.boot:spring-boot-starter-validation\"
    implementation \"io.jsonwebtoken:jjwt-api:0.11.5\"
    implementation \"com.github.ben-manes.caffeine:caffeine:2.9.0\"
}' > /home/gradle/project/build.gradle && \
mkdir -p /home/gradle/.gradle && \
echo 'systemProp.maven.repo.remote=http://192.168.0.124:8080/maven' > /home/gradle/.gradle/gradle.properties && \
mkdir -p /home/gradle/project && \
echo 'pluginManagement {
    repositories {
        maven {
            url \"http://192.168.0.124:8080/maven/\"
            allowInsecureProtocol = true
        }
        gradlePluginPortal()
        mavenCentral()
    }
}' > /home/gradle/project/settings.gradle && \
cd /home/gradle/project && gradle dependencies --refresh-dependencies"
```


package dep_test

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"what/internal/dep"
)

func TestGradle(t *testing.T) {
	fs := fstest.MapFS{
		"build.gradle": {Data: []byte(`
implementation 'org.apache.commons:commons-lang3:3.12.0'
implementation 'com.fasterxml.jackson.core:jackson-databind:2.12.5'
		`)},
	}

	m, err := dep.GetManager(dep.ManagerTypeJava, fs, ".")
	require.NoError(t, err)

	toFind := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"org.apache.commons:*", []dep.Dependency{{
			Vendor:  "org.apache.commons",
			Name:    "org.apache.commons:commons-lang3",
			Version: "3.12.0",
		}}},
	}
	for _, c := range toFind {
		deps, err := m.Find(c.pattern)
		require.NoError(t, err)
		assert.Equal(t, c.dependencies, deps)
	}

	toGet := []struct {
		name       string
		dependency dep.Dependency
		found      bool
	}{
		{"org.apache.commons:commons-lang3", dep.Dependency{
			Vendor:  "org.apache.commons",
			Name:    "org.apache.commons:commons-lang3",
			Version: "3.12.0",
		}, true},
		{"org.springframework.boot:spring-boot-maven-plugin", dep.Dependency{}, false},
	}
	for _, c := range toGet {
		d, ok := m.Get(c.name)
		assert.Equal(t, c.found, ok, c.name)
		assert.Equal(t, c.dependency, d, c.name)
	}
}

func TestMaven(t *testing.T) {
	fs := fstest.MapFS{
		"pom.xml": {Data: []byte(`
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
 xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
 <modelVersion>4.0.0</modelVersion>

 <groupId>com.example.test</groupId>
 <artifactId>test</artifactId>
 <version>0.0.1</version>

 <properties>
    <java.version>11</java.version>
</properties>

<parent>
    <groupId>org.springframework.boot</groupId>
    <artifactId>spring-boot-starter-parent</artifactId>
    <version>2.4.1</version>
    <relativePath/>
</parent>

<dependencies>
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-web</artifactId>
    </dependency>
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-data-jpa</artifactId>
    </dependency>
    <dependency>
        <groupId>mysql</groupId>
        <artifactId>mysql-connector-java</artifactId>
    </dependency>
    <dependency>
        <groupId>org.springframework.boot</groupId>
        <artifactId>spring-boot-starter-test</artifactId>
        <scope>test</scope>
    </dependency>
</dependencies>

<build>
    <finalName>spring-boot-maven-mysql</finalName>
    <plugins>
        <plugin>
            <groupId>org.springframework.boot</groupId>
            <artifactId>spring-boot-maven-plugin</artifactId>
        </plugin>
    </plugins>
</build>

<repositories>
    <repository>
        <id>oss.sonatype.org-snapshot</id>
        <url>https://oss.sonatype.org/content/repositories/snapshots</url>
    </repository>
</repositories>
</project>
		`)},
	}

	m, err := dep.GetManager(dep.ManagerTypeJava, fs, ".")
	require.NoError(t, err)

	toFind := []struct {
		pattern      string
		dependencies []dep.Dependency
	}{
		{"org.springframework.boot*", []dep.Dependency{
			{
				Vendor: "org.springframework.boot",
				Name:   "org.springframework.boot:spring-boot-starter-data-jpa",
			},
			{
				Vendor:  "org.springframework.boot",
				Name:    "org.springframework.boot:spring-boot-starter-parent",
				Version: "2.4.1",
			},
			{
				Vendor: "org.springframework.boot",
				Name:   "org.springframework.boot:spring-boot-starter-test",
			},
			{
				Vendor: "org.springframework.boot",
				Name:   "org.springframework.boot:spring-boot-starter-web",
			},
		}},
	}
	for _, c := range toFind {
		deps, err := m.Find(c.pattern)
		require.NoError(t, err)
		slices.SortFunc(deps, func(a, b dep.Dependency) int {
			return strings.Compare(a.Name, b.Name)
		})
		assert.EqualValues(t, c.dependencies, deps)
	}

	toGet := []struct {
		name       string
		dependency dep.Dependency
		found      bool
	}{
		{name: "org.apache.commons:commons-lang3"},
		{name: "org.springframework.boot:spring-boot-maven-plugin"},
	}
	for _, c := range toGet {
		d, ok := m.Get(c.name)
		assert.Equal(t, c.found, ok, c.name)
		assert.Equal(t, c.dependency, d, c.name)
	}
}

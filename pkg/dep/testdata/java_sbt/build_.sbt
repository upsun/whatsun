name := "play-sbt-example"

version := "1.0.0"

scalaVersion := "2.13.12"

lazy val root = (project in file("."))
  .enablePlugins(PlayScala)

libraryDependencies ++= Seq(
  guice,
  "com.typesafe.play" %% "play-akka-http-server" % "2.8.20",
  "com.typesafe.play" %% "play-json" % "2.9.4",
  "com.typesafe.play" %% "play-slick" % "5.1.0",
  "com.typesafe.play" %% "play-slick-evolutions" % "5.1.0",
  "org.postgresql" % "postgresql" % "42.6.0",
  "org.scalatestplus.play" %% "scalatestplus-play" % "5.1.0" % Test
)

resolvers += "Typesafe repository" at "https://repo.typesafe.com/typesafe/releases/" 
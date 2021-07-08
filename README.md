# Uptime Robot Exporter

A Prometheus exporter for Uptime Robot metrics.

## Getting started

You can either:

* download the sources and build the binary (Go required):

```
$ git clone https://github.com/eze-kiel/uptimerobot-exporter.git
$ cd uptimerobot-exporter/
$ make build
```

and then execute the binary at `./out/bin/uptimerobot-exporter`

* get the latest release [here](https://github.com/eze-kiel/uptimerobot-exporter/releases)

## Usage

```
Usage of uptimerobot-exporter:
  -api-key string
        Uptime Robot API key
  -inteval int
        Uptime robot API scrape interval, in seconds (default 30)
  -ip string
        IP on which the Prometheus server will be binded (default "0.0.0.0")
  -p string
        Port that will be used by the Prometheus server (default "9705")
```

Basically, you just have to pass your Uptime Robot API key. Of course, to avoid typing it in the terminal, you can provide it via an environment variable called `UPTIMEROBOT_API_KEY`.

## Docker

To use it with Docker, you can either:

* build your own image with the given Dockerfile:

```
$ docker build . -t uptimerobot-exporter:latest
```

* use the pre-built image (`ez3kiel/uptimerobot-exporter`).

In both cases, you have to provide the API key to run the container:

```
$ docker run -e UPTIMEROBOT_API_KEY=$(echo $UPTIMEROBOT_API_KEY) uptimerobot-exporter:latest
```

## Kubernetes

You can find the associated Helm charts [here](https://github.com/devops-works/helm-charts/tree/master/uptimerobot). You need to change `uptimerobot.apiKey` in `values.yaml` to make it working, or overwrite it with Helmfile.

## License

MIT
